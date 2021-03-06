package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-ini/ini"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tsubasaxZZZ/wutools/common"
)

var (
	kbnoOpt     = flag.String("n", "", "Specific KB NO(if you want to multiple, separate comma)")
	csvOpt      = flag.String("f", "", "(Not Implement)Specific CSV file")
	metaonlyOpt = flag.Bool("metadata-only", false, "If you want to get only metadata, specific this option")
	conOpt      = flag.Int("c", 10, "Specific max downloadconcurrent num(default:10)")
	daemonOpt   = flag.Bool("d", false, "Daemon mode")
	db          *sql.DB
)

func main() {
	// 引数のパース
	flag.Parse()

	if *daemonOpt {
		daemonize()
		return
	}
	if *kbnoOpt == "" {
		fmt.Println("You need specific KB no.(Please read --help)")
		return
	}
	strKbno := strings.Split(*kbnoOpt, ",")
	kbno := []int{}
	for _, v := range strKbno {
		si, err := strconv.Atoi(v)
		if err != nil {
			fmt.Print(err)
			return
		}
		kbno = append(kbno, si)
	}
	log.Printf("Target KB no:%v", kbno)

	// KB のリストの生成
	kbList := kb.NewKBList(kbno, *conOpt)

	log.Println(*kbList)

	// CSV へメタデータを出力
	kbList.ExportMetadataToCSV()

	// メタデータのみ取得のオプションがない場合にパッケージをダウンロード
	if !*metaonlyOpt {
		kbList.DownloadAllKB(*conOpt)
	}
}

func connectDB() error {
	// 設定ファイル読み込み
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Fail to read file: %v", err)
		return err
	}
	//user:password@tcp(host:port)/dbname
	connectionString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.Section("").Key("DATABASE_USERNAME").String(),
		cfg.Section("").Key("DATABASE_PASSWORD").String(),
		cfg.Section("").Key("DATABASE_SERVER").String(),
		cfg.Section("").Key("DATABASE_PORT").String(),
		cfg.Section("").Key("DATABASE_NAME").String(),
	)
	// DB 接続
	log.Printf("Connect mysql: %s", connectionString)
	db, err = sql.Open("mysql", connectionString)
	return err

}
func daemonize() {
	err := connectDB()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	//無限ループ
	for {
		// session テーブルのクエリ
		// 登録済み状態のもののみ取得
		log.Println("Query session table.")
		rows, err := db.Query(
			"SELECT id,kbno,sakey, saname, create_utc_date,update_utc_date,status FROM session WHERE `status` & ? = 1",
			kb.StatusRegistered,
		)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer rows.Close()

		// 行スキャン
		var sessions []kb.Session
		log.Println("Start scan rows.")
		for rows.Next() {
			var session kb.Session
			session.Db = db
			err := rows.Scan(
				&(session.ID),
				&(session.Kbno),
				&(session.Sakey),
				&(session.Saname),
				&(session.CreateDate),
				&(session.UpdateDate),
				&(session.Status),
			)
			if err != nil {
				log.Fatal(err.Error())
			}
			sessions = append(sessions, session)
		}
		if err := rows.Err(); err != nil {
			log.Panic(err.Error())
		}

		// KB単位で処理開始
		semaphore := make(chan int, 10)
		for _, session := range sessions {
			semaphore <- 1
			go session.ProcessSession()
			<-semaphore
		}

		cleanup()
		time.Sleep(10 * time.Second)
	}
}

func cleanup() {
	rows, err := db.Query(
		"SELECT id,kbno,sakey, saname, create_utc_date,update_utc_date,status FROM session WHERE `status` & ? != ?",
		kb.StatusCleanupComplete, kb.StatusCleanupComplete,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer rows.Close()

	// 行スキャン
	sessions := make(map[string][]kb.Session)
	for rows.Next() {
		var session kb.Session
		session.Db = db
		err := rows.Scan(
			&(session.ID),
			&(session.Kbno),
			&(session.Sakey),
			&(session.Saname),
			&(session.CreateDate),
			&(session.UpdateDate),
			&(session.Status),
		)
		if err != nil {
			log.Fatal(err.Error())
		}
		sessions[session.ID.String] = append(sessions[session.ID.String], session)
	}
	log.Printf("Start scan rows for cleanup.: cleanup session count=[%d]", len(sessions))
	if err := rows.Err(); err != nil {
		log.Panic(err.Error())
	}

	for id, sessionList := range sessions {
		canCleanup := true
		// 全てのパッケージがアップロード完了していたら削除可能
		for _, session := range sessionList {
			log.Printf("Session status check.: id=[%s], status=[%d]", id, session.Status)
			if session.Status != kb.StatusUploadComplete {
				canCleanup = false
			}
		}
		if canCleanup {
			log.Printf("Start cleanup: id=[%s]", id)
			err := os.RemoveAll(id)
			if err != nil {
				log.Printf("Cleanup error: id=[%s], error=[%v]", id, err.Error())
				continue
			}
			for _, session := range sessionList {
				session.ChangeStatus(kb.StatusCleanupComplete)
			}
			log.Printf("End cleanup: id=[%s]", id)
		}
	}

}
