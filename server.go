package main

//
// Website Server using Golang
//

import (
	"encoding/csv"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/render"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

//{ "_id" : ObjectId("557c8a526607852a166feda8"), "id" : 3031, "author" : "小山内薫", "author_en" : "OsanaiKaoru", "open_at" : "2001-05-01", "update_at" : "2014-09-17", "title" : "梨の実", "title_" : "なしのみ", "tag" : [ "", "", "", "", "", "", "", "", "", "" ], "cata" : [ "", "913" ], "cardlink" : "http://www.aozora.gr.jp/cards/000479/card3031.html", "booklink" : "http://www.aozora.gr.jp/cards/000479/files/3031_19531.html", "author_id" : 479, "load_at" : ISODate("2015-06-13T19:53:54.136Z") }
type Books struct {
	_id         bson.ObjectId `bson:"_id,omitempty" json:"id"`
	id          int           `bson:"id" json:"id"`
	title       string        `bson:"title" json:"title"`
	title_      string        `bson:"title_" json:"title_"`
	otitle      string        `bson:"otitle" json:"otitle"`
	tag         []string      `bson:"tag" json:"tag"`
	cata        []string      `bson:"cata" json:"cata"`
	cardlink    string        `bson:"cardlink" json:"cardlink"`
	booklink    string        `bson:"booklink" json:"booklink"`
	author_id   int           `bson:"author_id" json:"author_id"`
	author      string        `bson:"author" json:"author"`
	author_en   string        `bson:"author_en" json:"author_en"`
	open_at     string        `bson:"open_at" json:"open_at"`
	update_at   string        `bson:"update_at" json:"update_at"`
	load_at     time.Time     `bson:"load_at" json:"load_at"`
	extra       bson.M        `bson:",inline"`
	author_link string
}

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

const authorLinkPrefix = "http://www.aozora.gr.jp/index_pages/person"

type Page struct {
	Title string
	Data  string
}

// Search title/otitle/author for keyword and return author & book information
func GetBooksByKeyword(keyword string, c *mgo.Collection) (r []bson.M) {
	m := []bson.M{}
	if err := c.Find(bson.M{
		"$or": []interface{}{
			bson.M{"title": &bson.RegEx{Pattern: keyword, Options: "i"}},
			bson.M{"otitle": &bson.RegEx{Pattern: keyword, Options: "i"}},
			bson.M{"author": &bson.RegEx{Pattern: keyword, Options: "i"}},
		}}).Sort("author").All(&m); err == nil { // Do Query
		for _, v := range m {
			v["author_link"] = authorLinkPrefix + strconv.Itoa(v["author_id"].(int)) + ".html"
			m = append(m, v)
		}
	}

	m_ := ResultArray(m).CleanResult()
	return m_
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	m := martini.Classic()

	// render html templates from directory
	m.Use(render.Renderer())

	// Home
	m.Get("/", func(r render.Render) {
		ary := []Page{}
		p1 := Page{Title: "Kala", Data: "Google"}
		p2 := Page{Title: "Ashley", Data: "Tencent"}
		p3 := Page{Title: "Mama", Data: "Kaohsiung"}
		ary = append(ary, p1, p2, p3)
		r.HTML(200, "index", ary)
	})

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c := session.DB("aozora").C("books_go")

	m.Get("/book/", func(r render.Render) {
		// Open list file to get random author
		keyword := ""
		if f, ferr := os.Open("authorList.csv"); ferr != nil {
			panic(ferr)
		} else {
			// Read first line only
			reader := csv.NewReader(f)
			if ary, rerr := reader.Read(); rerr == nil {
				keyword = ary[rand.Int()%len(ary)]
			}
		}
		m_ := GetBooksByKeyword(keyword, c)
		r.HTML(200, "book", m_)
	})

	m.Get("/book/:str", func(params martini.Params, r render.Render) {
		m_ := GetBooksByKeyword(params["str"], c)
		r.HTML(200, "book", m_)
	})

	m.Post("/search", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/go/book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Run()
}
