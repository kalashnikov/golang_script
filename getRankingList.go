package main

import (
	"bytes"
	"fmt"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/css"
	html "github.com/moovweb/gokogiri/html"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const ranking_url = "http://www.aozora.gr.jp/access_ranking/"

type BookNode struct {
	rank           int
	author_name    string
	author_link    string
	book_id        int
	book_name      string
	book_link      string
	real_book_link string
	txt_link       string
}

func getDocByURL(url string) *html.HtmlDocument {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	h, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	doc, err := gokogiri.ParseHtml(h)
	if err != nil {
		panic(err)
	}

	return doc
}

func getRealLink(b BookNode, wg *sync.WaitGroup, c *mgo.Collection, s []*BookNode) {
	defer wg.Done()

	// Get the book id
	re := regexp.MustCompile(`card([0-9]+).html`)
	if array := re.FindStringSubmatch(b.book_link); len(array) == 2 {
		b.book_id, _ = strconv.Atoi(string(array[1]))
	}

	m := bson.M{}
	reid := regexp.MustCompile(`files/(.+).html`)
	if err := c.Find(bson.M{"id": b.book_id}).One(&m); err == nil { // Do Query
		b.real_book_link = m["booklink"].(string)
		if array := reid.FindStringSubmatch(b.real_book_link); len(array) == 2 {
			b.txt_link = "txt/" + string(array[1]) + ".txt"
		}
	}

	// No lock due to fix write position for each goroutine
	s[b.rank] = &b
}

func genRankList(latest_url string, c *mgo.Collection) string {
	// Parsing latest ranking page
	// Using Gokogiri and its CSS package
	var bookArray []BookNode
	doc := getDocByURL(latest_url)
	if nodeArr, err := doc.Search(css.Convert("tr td.normal a", css.GLOBAL)); err == nil {
		i, idx := 0, 0
		for i < len(nodeArr)-1 {
			// Use book link url to extract author_id and generate author link
			book_link := nodeArr[i].Attr("href")
			book_name := strings.TrimSpace(nodeArr[i].FirstChild().String())

			// Multiple author, need to step advance to find next book link
			for strings.Contains(book_link, "person") {
				i++
				book_link = nodeArr[i].Attr("href")
				book_name = strings.TrimSpace(nodeArr[i].FirstChild().String())
			}

			author_name := strings.TrimSpace(nodeArr[i+1].FirstChild().String())
			author_link := nodeArr[i+1].Attr("href")

			bn := BookNode{
				rank:        idx,
				author_name: author_name,
				author_link: author_link,
				book_name:   book_name,
				book_link:   book_link,
			}

			// Checking section
			//
			// aid := 0
			// re_author := regexp.MustCompile("person([0-9]+).html")
			// if array := re_author.FindStringSubmatch(author_link); len(array) == 2 {
			// 	aid, _ = strconv.Atoi(string(array[1]))
			// }
			// bid := 0
			// re_book := regexp.MustCompile("cards/([0-9]+)/card")
			// if array := re_book.FindStringSubmatch(book_link); len(array) == 2 {
			// 	bid, _ = strconv.Atoi(string(array[1]))
			// }
			// fmt.Printf("%s, %s | %d : %d | %t\n", author_link, book_link, aid, bid, (aid == bid))

			bookArray = append(bookArray, bn)

			i += 2
			idx += 1
		}
	}

	// Concurrent do it
	wg := &sync.WaitGroup{}
	slice := make([]*BookNode, len(bookArray))
	for _, n := range bookArray {
		wg.Add(1)
		go getRealLink(n, wg, c, slice)
	}
	wg.Wait()

	var markdown bytes.Buffer
	markdown.WriteString("### [青空文庫　アクセスランキング](" + latest_url + "):\n")
	for _, b := range slice {
		idx := strconv.Itoa(b.rank + 1)
		str := fmt.Sprintf("   %s. [%s](%s) - [%s](%s) [※](%s)\n", idx, b.author_name, b.author_link, b.book_name, b.real_book_link, b.txt_link)
		markdown.WriteString(str)
	}

	return markdown.String()
}

const staticName = "ranklist.md"
const work_folder = "/var/opt/www/go/"

func main() {

	err := os.Chdir(work_folder)
	if err != nil {
		panic(err)
	}

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c := session.DB("aozora").C("books_go")

	// Get lastest ranking html
	var latest_url string
	doc := getDocByURL(ranking_url)
	if nodeArr, err := doc.Search(css.Convert("a", css.GLOBAL)); err == nil {
		latest_url = ranking_url + nodeArr[0].Attr("href")
	}

	// Init output file name
	re_post := regexp.MustCompile("_xhtml.html")
	outputFile := fmt.Sprintf("ranklist_%s.md", re_post.ReplaceAllString(path.Base(latest_url), ""))

	if _, err = os.Stat(outputFile); err == nil {
		// No update. Return
		return
	} else {
		// Download and create new output file
		if out, err := os.Create(outputFile); err == nil {
			contents := genRankList(latest_url, c)
			out.WriteString(contents)
		}

		// Update the static link
		if _, err = os.Stat(staticName); err == nil {
			cmd := exec.Command("unlink", staticName)
			cmd.Run()
		}
		cmd := exec.Command("ln", "-s", outputFile, staticName)
		cmd.Run()
	}
}
