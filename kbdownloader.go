package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/tsubasaxZZZ/wutools/common"
)

var (
	kbnoOpt     = flag.String("n", "", "Specific KB NO(if you want to multiple, separate comma)")
	csvOpt      = flag.String("f", "", "(Not Implement)Specific CSV file")
	metaonlyOpt = flag.Bool("metadata-only", false, "If you want to get only metadata, specific this option")
	conOpt      = flag.Int("c", 10, "Specific max downloadconcurrent num(default:10)")
)

func main() {
	// 引数のパース
	flag.Parse()

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
