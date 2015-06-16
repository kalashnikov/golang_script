package main

import (
	"fmt"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/css"
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

func getRealLink(b BookNode, wg *sync.WaitGroup, s []*BookNode) {
	defer wg.Done()

	// Get the book id
	re := regexp.MustCompile(`card([0-9]+).html`)
	if array := re.FindStringSubmatch(b.book_link); len(array) == 2 {
		b.book_id, _ = strconv.Atoi(string(array[1]))
	}

	// TODO Use this Book id to get real booklink in mongodb

	s[b.rank] = &b
}

func main() {

	var latest_url string

	// Get lastest ranking html
	if resp, err := http.Get(ranking_url); err == nil {
		if html, err := ioutil.ReadAll(resp.Body); err == nil {
			if doc, err := gokogiri.ParseHtml(html); err == nil {
				if nodeArr, err := doc.Search(css.Convert("a", css.GLOBAL)); err == nil {
					latest_url = ranking_url + nodeArr[0].Attr("href")
				}
			}
		}
	}

	var bookArray []BookNode

	// Parsing latest ranking page
	// Using Gokogiri and its CSS package
	if resp, err := http.Get(latest_url); err == nil {
		if html, err := ioutil.ReadAll(resp.Body); err == nil {
			if doc, err := gokogiri.ParseHtml(html); err == nil {
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
			}
		}
	}

	// Concurrent do it
	wg := &sync.WaitGroup{}
	slice := make([]*BookNode, len(bookArray))
	for _, n := range bookArray {
		wg.Add(1)
		go getRealLink(n, wg, slice)
	}
	wg.Wait()

	for _, b := range slice {
		fmt.Println(strconv.Itoa(b.rank) + " | Author: " + b.author_name + " => " + b.author_link + " | Book[" + strconv.Itoa(b.book_id) + "]: " + b.book_name + " => " + b.book_link)
	}
}
