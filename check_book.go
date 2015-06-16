package main

import (
	"encoding/csv"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

//{ "_id" : ObjectId("557c8a526607852a166feda8"), "id" : 3031, "author" : "小山内薫", "author_en" : "OsanaiKaoru", "open_at" : "2001-05-01", "update_at" : "2014-09-17", "title" : "梨の実", "title_" : "なしのみ", "tag" : [ "", "", "", "", "", "", "", "", "", "" ], "cata" : [ "", "913" ], "cardlink" : "http://www.aozora.gr.jp/cards/000479/card3031.html", "booklink" : "http://www.aozora.gr.jp/cards/000479/files/3031_19531.html", "author_id" : 479, "load_at" : ISODate("2015-06-13T19:53:54.136Z") }
type Books struct {
	_id       bson.ObjectId `bson:"_id,omitempty" json:"id"`
	id        int           `bson:"id" json:"id"`
	title     string        `bson:"title" json:"title"`
	title_    string        `bson:"title_" json:"title_"`
	otitle    string        `bson:"otitle" json:"otitle"`
	tag       []string      `bson:"tag" json:"tag"`
	cata      []string      `bson:"cata" json:"cata"`
	cardlink  string        `bson:"cardlink" json:"cardlink"`
	booklink  string        `bson:"booklink" json:"booklink"`
	author_id int           `bson:"author_id" json:"author_id"`
	author    string        `bson:"author" json:"author"`
	author_en string        `bson:"author_en" json:"author_en"`
	open_at   string        `bson:"open_at" json:"open_at"`
	update_at string        `bson:"update_at" json:"update_at"`
	load_at   time.Time     `bson:"load_at" json:"load_at"`
	extra     bson.M        `bson:",inline"`
}

const zip_name string = "list_person_all_extended_utf8.zip"
const file_name string = "list_person_all_extended_utf8.csv"

// Download latest file
func prepareFile() {

	// Remove previous package
	if _, err := os.Stat(file_name); err == nil {
		err1 := os.Remove(zip_name)
		err2 := os.Remove(file_name)
		if err1 != nil || err2 != nil {
			fmt.Printf("Remove failed: %s\n", err1)
			panic(err1)
		} else {
			fmt.Println("Remove file finished.")
		}
	}

	// Download
	url_path := "http://www.aozora.gr.jp/index_pages/list_person_all_extended_utf8.zip"
	_, err := exec.Command("wget", url_path).Output()
	if err != nil {
		fmt.Printf("Download failed: %s\n", err)
		panic(err)
	} else {
		fmt.Println("Download finished.")
	}

	// Unzip
	if _, err = os.Stat(zip_name); err == nil {
		_, err = exec.Command("unzip", zip_name).Output()
		if err != nil {
			fmt.Printf("Unzip failed: %s\n", err)
			panic(err)
		} else {
			fmt.Println("Unzip finished.")
		}
	}
}

func main() {

	prepareFile()

	// Open File
	f, ferr := os.Open(file_name)
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}
	defer f.Close()

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("aozora").C("books_go")

	// Scan the file by line
	reader := csv.NewReader(f)
	_, _ = reader.Read() // Skip first
	for {
		ary, rerr := reader.Read()
		if rerr == io.EOF {
			break
		} else if rerr != nil {
			fmt.Printf("CSV Reader failed: %s\n", rerr)
			panic(rerr)
		}

		inc := 0 // Tweaking...

		did, iderr := strconv.Atoi(ary[0])
		if iderr != nil {
			fmt.Printf("ID aoit failed: %s\n", iderr)
			panic(iderr)
		}

		title, title_, otitle := ary[1], ary[2], ary[6]
		cata := strings.Split(strings.Replace(ary[8], "NDC", "", 1), " ")

		for !strings.Contains(ary[11+inc], "-") {
			inc += 1
		}

		// Get Author ID
		author_id, aiderr := strconv.Atoi(ary[14+inc])
		if aiderr != nil {
			fmt.Printf("Author ID aoit failed: %s\n", aiderr)
			panic(aiderr)
		}

		author, author_en := ary[15+inc]+ary[16+inc], ary[21+inc]+ary[22+inc]
		bookcard, booklink := ary[13+inc], ary[len(ary)-5]
		open_at, update_at, load_at := ary[11+inc], ary[12+inc], time.Now()

		_, err = c.Upsert(bson.M{"id": did}, bson.M{"$set": bson.M{
			"id":        did,
			"title":     title,
			"title_":    title_,
			"otitle":    otitle,
			"tag":       make([]string, 10),
			"cata":      cata,
			"cardlink":  bookcard,
			"booklink":  booklink,
			"author_id": author_id,
			"author":    author,
			"author_en": author_en,
			"open_at":   open_at,
			"update_at": update_at,
			"load_at":   load_at,
		},
		},
		)
		if err != nil {
			panic(err)
		}
	}

	// Check the result
	var m []bson.M
	_ = c.Find(nil).All(&m)
	for _, v := range m {
		fmt.Println(v)
	}
}
