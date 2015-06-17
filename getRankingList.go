package main

import (
	"bytes"
	"fmt"
	//"github.com/microcosm-cc/bluemonday"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/css"
	html "github.com/moovweb/gokogiri/html"
	//"github.com/russross/blackfriday"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
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
	if err := c.Find(bson.M{"id": b.book_id}).One(&m); err == nil { // Do Query
		b.real_book_link = m["booklink"].(string)
	}

	// No lock due to fix write position for each goroutine
	s[b.rank] = &b
}

func main() {

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

	// Parsing latest ranking page
	// Using Gokogiri and its CSS package
	var bookArray []BookNode
	doc = getDocByURL(latest_url)
	if nodeArr, err := doc.Search(css.Convert("tr td.normal a", css.GLOBAL)); err == nil {
		for i := 0; i < len(nodeArr)-1; i += 2 {
			author_name := strings.TrimSpace(nodeArr[i+1].FirstChild().String())
			author_link := nodeArr[i+1].Attr("href")

			// Use book link url to extract author_id and generate author link
			book_link := nodeArr[i].Attr("href")
			book_name := strings.TrimSpace(nodeArr[i].FirstChild().String())

			if strings.Contains(book_link, "person") {
				author_name, book_name = book_name, author_name
				author_link, book_link = book_link, author_link
			}

			bn := BookNode{
				rank:        i / 2,
				author_name: author_name,
				author_link: author_link,
				book_name:   book_name,
				book_link:   book_link,
			}
			bookArray = append(bookArray, bn)
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
		str := fmt.Sprintf("   %s. [%s](%s) - [%s](%s)\n", idx, b.author_name, b.author_link, b.book_name, b.real_book_link)
		markdown.WriteString(str)
	}
	fmt.Println(markdown.String())

	// Markdown to HTML by using Blackfriday
	//unsafe := blackfriday.MarkdownCommon(markdown.Bytes())
	//html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	//fmt.Println(string(html))
}
