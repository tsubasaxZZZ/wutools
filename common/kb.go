package kb

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type KBList struct {
	kbs []KB
}

type KB struct {
	no             int
	title          string
	kbPackageInfos []KBPackageInfo
}

type KBPackageInfo struct {
	packageTitle string
	downloadLink string
	architecture string
	fileName     string
	language     string
	fileSize     int64
}

const (
	catalogURL        = "https://www.catalog.update.microsoft.com/Search.aspx?q=%d"
	downloadDialogURL = "https://www.catalog.update.microsoft.com/DownloadDialog.aspx"
)

// ExportMetadataToCSV : メタデータを CSV にエクスポートする
func (kbList KBList) ExportMetadataToCSV() error {
	file, err := os.Create("metadata.csv")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"KB", "Title(NotImpl)", "PackageTitle", "Architecture", "Filename", "Language", "Filesize(bytes)", "Packagelink"})
	for _, kb := range kbList.kbs {
		for _, pkg := range kb.kbPackageInfos {
			writer.Write([]string{strconv.Itoa(kb.no), kb.title, pkg.packageTitle, pkg.architecture, pkg.fileName, pkg.language, strconv.FormatInt(pkg.fileSize, 10), pkg.downloadLink})
		}
	}
	writer.Flush()
	return nil
}

// DownloadAllKB : ファイルのダウンロード
func (kbList KBList) DownloadAllKB(maxConcurrent int) error {
	ch := make(chan KB, len(kbList.kbs))
	wg := &sync.WaitGroup{}
	semaphore := make(chan int, maxConcurrent)

	for _, kb := range kbList.kbs {
		wg.Add(1)
		go func(kb KB, ch chan KB) {
			log.Printf("--------------- start download all package : kb=[%d]", kb.no)
			defer wg.Done()
			for _, kbPackageInfo := range kb.kbPackageInfos {
				semaphore <- 1
				// ファイルの存在チェック
				// ファイルが存在する場合は処理をスキップ(1つのKBで、複数OS分のパッケージがリストされている場合、ファイルが同一の場合がある)
				if _, err := os.Stat(kbPackageInfo.fileName); err == nil {
					log.Printf("file is exists. skip.. : kb=[%d], fileName=[%s]", kb.no, kbPackageInfo.fileName)
					<-semaphore
					continue
				}

				log.Printf("start download KB-Pkg : kb=[%d], fileName=[%s]", kb.no, kbPackageInfo.fileName)
				resp, err := http.Get(kbPackageInfo.downloadLink)
				if err != nil {
					log.Fatal(err)
					<-semaphore
					continue
				}
				defer resp.Body.Close()
				file, err := os.Create(kbPackageInfo.fileName)
				if err != nil {
					log.Fatal(err)
					<-semaphore
					continue
				}
				defer file.Close()

				io.Copy(file, resp.Body)
				log.Printf("end download KB-Pkg : kb=[%d], fileName=[%s]", kb.no, kbPackageInfo.fileName)
				<-semaphore
			}
			log.Printf("end download KB : kb=[%d]", kb.no)
			ch <- kb
		}(kb, ch)
	}
	//wg.Wait()
	//close(ch)
	for range kbList.kbs {
		log.Printf("--------------- end download all package : kb=[%d]", (<-ch).no)
	}
	close(ch)
	return nil
}

// NewKBList : KB番号から、URLやタイトルのリストを生成する
func NewKBList(nos []int) *KBList {
	kbList := new(KBList)
	ch := make(chan KB, len(nos))

	wg := &sync.WaitGroup{}
	// CPU 数で最大のスレッド数を制限
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	semaphore := make(chan int, cpus)
	for _, no := range nos {
		wg.Add(1)
		go func(no int, ch chan KB) {
			defer wg.Done()
			semaphore <- 1
			buildKB(no, ch)
			<-semaphore
		}(no, ch)
	}
	wg.Wait()
	close(ch)
	for kb := range ch {
		kbList.kbs = append(kbList.kbs, kb)
	}
	return kbList
}
func buildKB(no int, ch chan KB) {
	kb := &KB{no: no}

	doc, err := goquery.NewDocument(fmt.Sprintf(catalogURL, kb.no))
	if err != nil {
		log.Fatal("url scarapping failed")
	}

	//抜き出してくる文字列:
	//<a id="ef673d9c-0e61-412b-be87-9eba39fe13dd_link" href="javascript:void(0);" onclick="goToDetails(";ef673d9c-0e61-412b-be87-9eba39fe13dd");">
	doc.Find("tbody > tr > td > a").Each(
		func(_ int, s *goquery.Selection) {
			kbPackageInfo := KBPackageInfo{}

			onclick, ok := s.Attr("onclick")
			if ok && strings.Contains(onclick, "goToDetails") {
				// goToDetails の ID 部分だけ取得
				updateID := strings.Replace(
					strings.Replace(onclick, "goToDetails(\"", "", -1), "\");",
					"",
					-1,
				)
				kbPackageInfo.packageTitle = strings.TrimSpace(s.Text())
				log.Printf("Get Package title and Id:packageTitle=[%s], onclick=[%s], updateID=[%s]", kbPackageInfo.packageTitle, onclick, updateID)

				//----------------------------------
				// scraiping package download link
				//----------------------------------
				// Request
				data := url.Values{}
				data.Set("updateIDs", fmt.Sprintf(`[{"size":0,"languages":"","uidInfo":"%s","updateID":"%s"}]`, updateID, updateID))
				req, err := http.NewRequest(
					"POST",
					downloadDialogURL,
					strings.NewReader(data.Encode()),
				)
				if err != nil {
					log.Fatal(err)
				}
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				client := &http.Client{}

				// Response
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(err)
				}

				//----------------------------------
				// scraiping for download dialog
				//----------------------------------
				body, _ := ioutil.ReadAll(resp.Body)
				dialogBodyDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
				if err != nil {
					log.Fatal("url scarapping failed")
				}
				html, _ := dialogBodyDoc.Html()
				r := regexp.MustCompile(`downloadInformation\[0\]\.files\[0\]\.(\S+) = '(\S+)';`)
				m := map[string]string{}
				for _, v := range r.FindAllStringSubmatch(html, -1) {
					m[v[1]] = v[2]
				}
				log.Printf("Get file information: m=%s", m)
				defer resp.Body.Close()

				kbPackageInfo.downloadLink = m["url"]
				kbPackageInfo.architecture = m["architectures"]
				kbPackageInfo.fileName = m["fileName"]
				kbPackageInfo.language = m["longLanguages"]
				// ファイルサイズの取得(HEAD)
				res, err := http.Head(kbPackageInfo.downloadLink)
				if err != nil {
					log.Fatal("HEAD file error")
				}
				kbPackageInfo.fileSize = res.ContentLength
				kb.kbPackageInfos = append(kb.kbPackageInfos, kbPackageInfo)
			}

		})
	ch <- *kb
}
