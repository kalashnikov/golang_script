package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const path = "/home/kalaexj/book_txt/"

func main() {
	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c := session.DB("aozora").C("books_go")

	m := []bson.M{}
	books := make(map[string]bool)
	if err := c.Find(nil).All(&m); err == nil { // Do Query
		for _, v := range m {
			books[v["title"].(string)] = false
		}
	}

	re1 := regexp.MustCompile(".txt")
	re2 := regexp.MustCompile("(校正.*)")

	nnovel, duplicate := 0, 0

	os.Chdir(path)
	if a, err := filepath.Glob("*.txt"); err == nil {
		for _, file := range a {
			fname := re1.ReplaceAllString(file, "")
			str := re2.ReplaceAllString(fname, " ")

			author, title := "", ""
			if strings.Contains(str, "－") {
				ary := strings.Split(str, "－")
				author, title = ary[0], ary[1]
			} else if strings.Contains(str, "-") {
				ary := strings.Split(str, "-")
				author, title = ary[0], ary[1]
			} else if strings.Contains(str, " ") {
				idx := strings.Index(str, " ")
				author, title = str[0:idx-1], str[idx+1:]
			} else if strings.Contains(str, "_") {
				idx := strings.LastIndex(str, "_")
				author, title = str[0:idx-1], str[idx+1:]
			} else {
				panic(str)
			}

			find := false
			if _, ok := books[title]; ok {
				find = true
				duplicate++
			} else {
				nnovel++
			}

			if strings.Contains(title, "(") {
				title = title[0 : len(title)-2]
			}
			fmt.Printf("[%t] %s : %s | %s \n", find, author, title, file)
		}
		fmt.Printf("### Total: %d | d:%d\n", nnovel, duplicate)
	}
}
