package kb

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// StatusRegistered : 登録済み(開始前)
	StatusRegistered = 0x1
	// StatusMetadataInprogress : メタデータ取得中
	StatusMetadataInprogress = 0x2
	// StautsMetadataComplete : メタデータ取得完了
	StautsMetadataComplete = 0x4
	// StatusDownloadInprogress : ダウンロード中
	StatusDownloadInprogress = 0x8
	// StatusDownloadComplete : ダウンロード完了
	StatusDownloadComplete = 0x10
	// StatusUploadInprogress ファイルのアップロード中
	StatusUploadInprogress = 0x20
	// StatusUploadComplete ファイルのアップロード完了
	StatusUploadComplete = 0x40
	// StatusDownloadSkip ダウンロードのスキップ
	StatusDownloadSkip = 0x80
	// StatusError エラー
	StatusError = 0x100
)

type Session struct {
	ID         sql.NullString
	Kbno       int
	Sakey      sql.NullString
	CreateDate time.Time
	UpdateDate time.Time
	Status     int
	Db         *sql.DB
}

func (session *Session) changeStatus(toStatus int) {

	log.Printf("Change session status: id=[%s], kbno=[%d], from-status=[%d], to-status=[%d]", session.ID.String, session.Kbno, session.Status, toStatus)
	_, err := session.Db.Exec(
		"UPDATE session SET status = ?, update_utc_date=? WHERE id = ? AND kbno = ?",
		toStatus, time.Now(), session.ID, session.Kbno,
	)
	if err != nil {
		log.Printf(err.Error())
	}
	session.Status = toStatus
	log.Printf("Change session status complete: id=[%s], kbno=[%d]",
		session.ID.String, session.Kbno)

}

func (packageInfo *PackageInfo) changeStatusPackageInfo(session Session, toStatus int) {
	log.Printf("Change packageInfo status: id=[%s], kbno=[%d], pkg-name=[%s], from-status=[%d], to-status=[%d]",
		session.ID.String, session.Kbno, packageInfo.FileName, packageInfo.Status, toStatus)
	_, err := session.Db.Exec(
		"UPDATE package SET status = ?, update_utc_date=? WHERE session_id = ? AND title = ?",
		toStatus, time.Now(), session.ID, packageInfo.Title,
	)
	if err != nil {
		log.Printf(err.Error())
	}
	packageInfo.Status = toStatus
	log.Printf("Change packageInfo status complete: id=[%s], kbno=[%d], pkg-name=[%s]",
		session.ID.String, session.Kbno, packageInfo.FileName)

}

func (session Session) ProcessSession() {

	// 処理開始
	log.Printf("Start process session: id=[%s], kbno=[%d], status=[%d]\n", session.ID.String, session.Kbno, session.Status)

	// ステータスをメタデータ取得中に変更
	session.changeStatus(StatusMetadataInprogress)

	// KB 情報の取得
	kbinfo := BuildKBInfo(session.Kbno)
	log.Printf("Complete get KB information: id=[%s], kbinfo=[%+v]", session.ID.String, kbinfo)

	// KB 情報をデータベースに格納
	log.Printf("INSERT package information: id=[%s], kbno=[%d]", session.ID.String, session.Kbno)
	for _, p := range kbinfo.PackageInfos {
		_, err := session.Db.Exec(
			"INSERT INTO package(session_id, kbno, title, downloadlink, architecture, fileName, language, fileSize, create_utc_date, update_utc_date, status) VALUES(?,?,?,?,?,?,?,?,?,?,?)",
			session.ID, session.Kbno, p.Title, p.DownloadLink, p.Architecture, p.FileName, p.Language, p.FileSize, time.Now(), time.Now(), StautsMetadataComplete,
		)
		if err != nil {
			log.Printf("INSERT ERROR: id=[%s], kbno=[%d]\n", session.ID.String, session.Kbno)
		}
		p.Status = StautsMetadataComplete
	}

	// ステータスをメタデータ取得完了に変更
	session.changeStatus(StautsMetadataComplete)

	//----------------------------
	// SAキーがある場合ダウンロード
	//----------------------------
	// ステータスをダウンロード中に変更
	session.changeStatus(StatusDownloadInprogress)
	// ファイルのダウンロード
	for _, kbPackageInfo := range kbinfo.PackageInfos {
		// packageのステータス変更
		kbPackageInfo.changeStatusPackageInfo(session, StatusDownloadInprogress)
		// ディレクトリが存在しない場合はディレクトリを作成
		if err := os.Mkdir(session.ID.String, 0777); err != nil {
			log.Printf("Directory is already exists.: id=[%s], kbno=[%d], error=[%s]", session.ID.String, session.Kbno, err.Error())
		}

		filePath := filepath.Join(session.ID.String, kbPackageInfo.FileName)

		err := func() error {

			// ファイルの存在チェック
			// ファイルが存在する場合は処理をスキップ(1つのKBで、複数OS分のパッケージがリストされている場合、ファイルが同一の場合がある)
			if _, err := os.Stat(filePath); err == nil {
				log.Printf("file is exists. skip.. : kb=[%d], fileName=[%s]", session.Kbno, filePath)
				kbPackageInfo.changeStatusPackageInfo(session, StatusDownloadSkip)
				return nil
			}

			log.Printf("start download KB-Pkg : kb=[%d], fileName=[%s], filePath=[%s]", session.Kbno, kbPackageInfo.FileName, filePath)
			resp, err := http.Get(kbPackageInfo.DownloadLink)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			io.Copy(file, resp.Body)
			log.Printf("end download KB-Pkg : kb=[%d], fileName=[%s]", session.Kbno, kbPackageInfo.FileName)
			// packageのステータス変更
			kbPackageInfo.changeStatusPackageInfo(session, StatusDownloadComplete)

			return nil
		}()
		if err != nil {
			kbPackageInfo.Status = StatusError
			log.Print(err)
			continue
		}
		// ハッシュの計算
		hash, err := hashFileMd5(filePath)
		if err != nil {
			log.Printf("Hash couldn't get : kb=[%d], fileName=[%s]", session.Kbno, kbPackageInfo.FileName)
			kbPackageInfo.Status = StatusError
			continue
		}
		kbPackageInfo.MD5hash = hash
		log.Printf("Culculated Hash : kb=[%d], fileName=[%s], hash=[%s]", session.Kbno, kbPackageInfo.FileName, kbPackageInfo.MD5hash)
		// Storage Account へアップロード

		// ハッシュの取得と比較

	}
	// ディレクトリの削除

	// ステータスをダウンロード完了に変更
	session.changeStatus(StatusDownloadComplete)

	// 処理終了
	log.Printf("End process session: id=[%s], kbno=[%d], status=[%d]\n", session.ID.String, session.Kbno, session.Status)

}

func hashFileMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}
