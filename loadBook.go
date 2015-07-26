package main

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c := session.DB("aozora").C("books_go")

	// Get current title list
	m := []bson.M{}
	books := make(map[string]bool)
	if err := c.Find(nil).All(&m); err == nil { // Do Query
		for _, v := range m {
			books[v["title"].(string)] = false
		}
	}

	fname := "fnameList.csv"

	// Open File
	f, ferr := os.Open(fname)
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	s, e := Readln(r)
	for e == nil {
		ary := strings.Split(s, ",")

		if id, err := strconv.Atoi(ary[0]); err == nil && id > 1000000 {
			author, title := GetTitle(ary[1])
			if _, ok := books[title]; !ok {
				fmt.Printf("%d | %s : %s\n", id, author, title)
				_, err = c.Upsert(bson.M{"id": id}, bson.M{"$set": bson.M{
					"id":        id,
					"title":     title,
					"title_":    title,
					"otitle":    title,
					"tag":       make([]string, 10),
					"cata":      make([]string, 3),
					"cardlink":  "",
					"booklink":  ary[1],
					"txtlink":   ary[1],
					"author_id": 0,
					"author":    author,
					"author_en": author,
					"open_at":   time.Now(),
					"update_at": time.Now(),
					"load_at":   time.Now(),
				},
				},
				)
				if err != nil {
					panic(err)
				}
			}
		}
		s, e = Readln(r)
	}
}

// Readln returns a single line (without the ending \n)
// from the input buffered reader.
// An error is returned iff there is an error with the
// buffered reader.
func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func GetTitle(file string) (string, string) {
	if strings.Contains(file, "/") {
		file = path.Base(file)
	}

	re1 := regexp.MustCompile(".txt")
	re2 := regexp.MustCompile("(校正.*)")

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

	if strings.Contains(title, "(") {
		title = title[0 : len(title)-2]
	}
	return author, title
}
