package main

import (
	"fmt"
	"github.com/bluele/mecab-golang"
	"golang.org/x/text/width"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

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

const authorLinkPrefix = "http://www.aozora.gr.jp/index_pages/person"
const RETURN_MAX_LENGTH = 20

func GetBookByList(docList []int, c *mgo.Collection) (r []bson.M) {
	m := []bson.M{}
	if err := c.Find(bson.M{"id": bson.M{"$in": docList}}).All(&m); err == nil { // Do Query
		for _, v := range m {
			v["author_link"] = authorLinkPrefix + strconv.Itoa(v["author_id"].(int)) + ".html"
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

// Search words using TF-IDF and return author & book information based on scores
func GetBooksByWords(keyword []string, c *mgo.Collection) []int {
	m := bson.M{}
	results := map[int]float64{} // Book ID to TF-IDF scores
	for _, word := range keyword {
		if err := c.Find(bson.M{"word": word}).One(&m); err == nil { // Do Query
			//http://golang.org/pkg/reflect/
			// http://stackoverflow.com/questions/14025833/range-over-interface-which-stores-a-slice
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

	//for i, v := range final {
	//	fmt.Printf("%d(%g) ", v, vs.vals[i])
	//}
	//fmt.Println("\n")
	//fmt.Println(final)
	return final
}

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

func main() {

	str := "京都　散策"

	stopwords := getStopWords()
	words := cleanWords(parseToNode(str), stopwords)

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c1 := session.DB("aozora").C("tf_idf")
	c2 := session.DB("aozora").C("books_go")

	list := GetBooksByWords(words, c1)
	m := GetBookByList(list, c2)

	idx := 0
	for _, v := range m {
		fmt.Printf("%d: %s - %s\n", idx, v["author"], v["title"])
		idx++
	}
}
