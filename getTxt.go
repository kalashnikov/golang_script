package main

//
// Get Aozora HTML file from website
//

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// For Sorting
type ResultArray []bson.M

func (a ResultArray) Len() int           { return len(a) }
func (a ResultArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ResultArray) Less(i, j int) bool { return a[i]["author"].(string) < a[j]["author"].(string) }

// Clean the result of MongoDB query
func (a ResultArray) CleanResult() ResultArray {
	var re ResultArray
	m := make(map[string]bool)

	// Sort and remove duplicate
	sort.Sort(a)
	for _, i := range a {
		key := i["author"].(string) + i["title"].(string)
		if _, ok := m[key]; !ok {
			re = append(re, i)
			m[key] = true
		}
	}
	return re
}

func download_file(url string) {
	fmt.Println(url)
	resp, _ := http.Get(url)

	filename := filepath.Base(url)
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	io.Copy(out, resp.Body)
	resp.Body.Close()
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

	// Get all the booklink from DB
	m := []bson.M{}
	urlArray := make([]string, 15000)
	if err := c.Find(nil).All(&m); err == nil { // Do Query
		m_ := ResultArray(m).CleanResult()
		for _, v := range m_ {
			urlArray = append(urlArray, v["booklink"].(string))
		}
	}

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			for url := range tasks {
				download_file(url)
			}
			wg.Done()
		}()
	}

	for _, n := range urlArray {
		if n == "" || !strings.Contains(n, "aozora.gr.jp") {
			continue
		}

		// Add the url to task list
		tasks <- n
	}
	close(tasks)

	wg.Wait()
}
