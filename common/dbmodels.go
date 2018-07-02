package kb

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
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

	log.Printf("Change status: id=[%s], kbno=[%d], from-status=[%d], to-status=[%d]", session.ID.String, session.Kbno, session.Status, toStatus)
	_, err := session.Db.Exec(
		"UPDATE session SET status = ?, update_utc_date=? WHERE id = ? AND kbno = ?",
		toStatus, time.Now(), session.ID, session.Kbno,
	)
	if err != nil {
		log.Printf(err.Error())
	}
	log.Printf("Change status complete: id=[%s], kbno=[%d], from-status=[%d], to-status=[%d]", session.ID.String, session.Kbno, session.Status, toStatus)
	session.Status = toStatus

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
			log.Printf("INSERT ERROR: id=[%s], kbno=[%d]", session.ID.String, session.Kbno)
		}
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
		err := func() error {
			// ディレクトリが存在しない場合はディレクトリを作成

			// ファイルの存在チェック
			// ファイルが存在する場合は処理をスキップ(1つのKBで、複数OS分のパッケージがリストされている場合、ファイルが同一の場合がある)
			if _, err := os.Stat(kbPackageInfo.FileName); err == nil {
				log.Printf("file is exists. skip.. : kb=[%d], fileName=[%s]", session.Kbno, kbPackageInfo.FileName)
				return err
			}

			log.Printf("start download KB-Pkg : kb=[%d], fileName=[%s]", session.Kbno, kbPackageInfo.FileName)
			resp, err := http.Get(kbPackageInfo.DownloadLink)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			file, err := os.Create(kbPackageInfo.FileName)
			if err != nil {
				return err
			}
			defer file.Close()

			io.Copy(file, resp.Body)
			log.Printf("end download KB-Pkg : kb=[%d], fileName=[%s]", session.Kbno, kbPackageInfo.FileName)

			// ハッシュの計算

			// Storage Account へアップロード

			// ハッシュの取得と比較

			// ディレクトリの削除
			return nil
		}()
		if err != nil {
			log.Print(err)
		}

	}
	// SA にアップロード
	// ステータスをダウンロード完了に変更
	session.changeStatus(StatusDownloadComplete)

	// 処理終了
	log.Printf("End process session: id=[%s], kbno=[%d], status=[%d]\n", session.ID.String, session.Kbno, session.Status)

}
