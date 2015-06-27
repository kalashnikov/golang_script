package main

//
// Website Server using Golang
//

import (
	"encoding/csv"
	"github.com/bluele/mecab-golang"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"golang.org/x/text/width"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const AUTHORLINKPREFIX = "http://www.aozora.gr.jp/index_pages/person"
const RETURN_MAX_LENGTH = 30

// https://gist.github.com/kylelemons/1236125
type ValSorter struct {
	keys []int
	vals []float64
}

func newvalsorter(m map[int]float64) *ValSorter {
	vs := &ValSorter{
		keys: make([]int, 0, len(m)),
		vals: make([]float64, 0, len(m)),
	}
	for k, v := range m {
		vs.keys = append(vs.keys, k)
		vs.vals = append(vs.vals, v)
	}
	return vs
}

func (vs *ValSorter) Sort() {
	sort.Sort(vs)
}

func (vs *ValSorter) Len() int           { return len(vs.vals) }
func (vs *ValSorter) Less(i, j int) bool { return vs.vals[i] > vs.vals[j] }
func (vs *ValSorter) Swap(i, j int) {
	vs.vals[i], vs.vals[j] = vs.vals[j], vs.vals[i]
	vs.keys[i], vs.keys[j] = vs.keys[j], vs.keys[i]
}

func getStopWords() map[string]bool {
	// Create set of stopwords
	f, err := ioutil.ReadFile("stopwords.csv")
	if err != nil {
		panic(err)
	}
	ary := strings.Split(string(f), ",")
	stopwords := make(map[string]bool, len(ary))
	for _, v := range ary {
		stopwords[v] = true
	}
	return stopwords
}

func filter(word string) (string, bool) {
	str := strings.ToLower(word)
	str = width.Narrow.String(str)
	str = strings.TrimSpace(str)

	// Use unicode method to check the word is meaningful or not
	// There exist many Symbol or non-sense words ...
	isWord := false
	runes := []rune(str)
	for _, u := range runes {
		if unicode.IsNumber(u) || unicode.IsLetter(u) {
			isWord = true
			break
		}
	}

	return str, isWord
}

func cleanWords(ary []string, stopwords map[string]bool) []string {
	out := make([]string, 0, len(ary))
	set := make(map[string]bool, len(ary)/10)
	for _, v := range ary {
		if str, isWord := filter(v); str != "" && isWord && !stopwords[str] && !set[str] {
			set[str] = true
			out = append(out, v) // Original word
		}
	}
	return out
}

func parseToNode(contents string) []string {

	// Init mecab
	m, err := mecab.New("-Owakati")
	if err != nil {
		panic(err)
	}
	defer m.Destroy()

	tg, err := m.NewTagger()
	if err != nil {
		panic(err)
	}
	defer tg.Destroy()

	output := make([]string, 50)
	lt, err := m.NewLattice(contents)
	if err != nil {
		panic(err)
	}
	defer lt.Destroy()

	node := tg.ParseToNode(lt)
	for {
		features := strings.Split(node.Feature(), ",")
		if features[0] == "名詞" {
			output = append(output, node.Surface())
		}
		if node.Next() != nil {
			break
		}
	}
	return output
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
			v["author_link"] = AUTHORLINKPREFIX + strconv.Itoa(v["author_id"].(int)) + ".html"
			m = append(m, v)
		}
	}

	m_ := ResultArray(m).CleanResult()
	return m_
}

func GetBookByList(docList []int, c *mgo.Collection) (r []bson.M) {
	m := []bson.M{}
	if err := c.Find(bson.M{"id": bson.M{"$in": docList}}).All(&m); err == nil { // Do Query
		for _, v := range m {
			v["author_link"] = AUTHORLINKPREFIX + strconv.Itoa(v["author_id"].(int)) + ".html"
			m = append(m, v)
		}
	}

	m_ := ResultArray(m).CleanResult()

	// Sort into same order of docList
	// O(n^2) but only size=20....
	m__ := []bson.M{}
	for _, v := range docList {
		for _, d := range m_ {
			if v == d["id"].(int) {
				m__ = append(m__, d)
			}
		}
	}

	return m__
}

// Search words using TF-IDF and return book id list sorted by scores
func GetBooksByWords(keyword []string, c *mgo.Collection) []int {
	m := bson.M{}
	results := map[int]float64{} // Book ID to TF-IDF scores
	for _, word := range keyword {
		if err := c.Find(bson.M{"word": word}).One(&m); err == nil { // Do Query
			docs := reflect.ValueOf(m["docs"])
			scores := reflect.ValueOf(m["score"])
			for i := 0; i < docs.Len(); i++ {
				idx := docs.Index(i).Interface().(int)
				if score, ok := results[idx]; ok {
					score += scores.Index(i).Interface().(float64)
				} else {
					results[idx] = scores.Index(i).Interface().(float64)
				}
			}
		}
	}

	// Sort by Scores
	vs := newvalsorter(results)
	vs.Sort()

	// Limit by RETURN_MAX_LENGTH
	var final []int
	if len(vs.keys) > RETURN_MAX_LENGTH {
		final = vs.keys[:RETURN_MAX_LENGTH-1]
	} else {
		final = vs.keys
	}
	return final
}

type TemplateBag struct {
	Title string
	Msg   string
	Ary   ResultArray
}

func main() {

	rand.Seed(time.Now().UTC().UnixNano())

	// Stop word list
	stopwords := getStopWords()

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c_book := session.DB("aozora").C("books_go")
	c_score := session.DB("aozora").C("tf_idf")

	m := martini.Classic()

	// render html templates from directory
	m.Use(render.Renderer())

	// Home
	m.Get("/", func(r render.Render) {
		ary := []TemplateBag{}
		p1 := TemplateBag{Title: "Kala", Msg: "Google"}
		p2 := TemplateBag{Title: "Ashley", Msg: "Tencent"}
		p3 := TemplateBag{Title: "Mama", Msg: "Kaohsiung"}
		ary = append(ary, p1, p2, p3)
		r.HTML(200, "index", ary)
	})

	m.Get("/book/", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		if _, err := os.Stat("/var/opt/www/go/ranklist.md"); err == nil {
			if b, err := ioutil.ReadFile("ranklist.md"); err == nil {
				re.HTML(200, "rank", string(b))
			}
		} else {
			url := "/go/book/random"
			http.Redirect(w, r, url, 302)
		}
	})

	m.Get("/book/:str", func(params martini.Params, r render.Render) {
		keyword := params["str"]
		if keyword == "random" {
			if f, ferr := os.Open("authorList.csv"); ferr != nil {
				panic(ferr)
			} else {
				// Read first line only
				reader := csv.NewReader(f)
				if ary, rerr := reader.Read(); rerr == nil {
					keyword = ary[rand.Int()%len(ary)]
				}
			}
		}
		m_ := GetBooksByKeyword(keyword, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		r.HTML(200, "book", bag)
	})

	m.Get("/search-book/", func(w http.ResponseWriter, r *http.Request, re render.Render) {
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
		m_ := GetBooksByKeyword(keyword, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "search", bag)
	})

	m.Get("/search-book/:str", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		keyword := params["str"]
		words := cleanWords(parseToNode(keyword), stopwords)
		list := GetBooksByWords(words, c_score) // Get Book list by score
		m_ := GetBookByList(list, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "search", bag)
	})

	m.Post("/search", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/go/book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Post("/search-book", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/go/search-book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Run()
}
