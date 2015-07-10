package book

import (
	"github.com/bluele/mecab-golang"
	"github.com/garyburd/redigo/redis"
	"github.com/qiniu/iconv"
	"golang.org/x/text/width"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const local_path = "/home/kalaexj/git-repo/golang_script/txtraw/"
const html_path = "/home/kalaexj/git-repo/golang_script/html/"
const AUTHORLINKPREFIX = "http://www.aozora.gr.jp/index_pages/person"
const work_folder = "/var/opt/www/go/"
const RETURN_MAX_LENGTH = 30

// a pool embedding the original pool and adding adbno state
type DbnoPool struct {
	redis.Pool
}

// "overriding" the Get method
func (p *DbnoPool) Get(dbId int) redis.Conn {
	conn := p.Pool.Get()
	conn.Do("SELECT", dbId)
	return conn
}

func InitRedisPool() DbnoPool {
	pool2 := DbnoPool{
		redis.Pool{
			MaxIdle:   80,
			MaxActive: 12000, // max number of connections
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", ":6379")
				if err != nil {
					panic(err.Error())
				}
				return c, err
			},
		},
	}
	return pool2
}

// Create a structure from map and sorted by value
// https://gist.github.com/kylelemons/1236125
type ValSorter struct {
	keys []int
	vals []float64
}

func NewValSorter(m map[int]float64) *ValSorter {
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

func GetStopWords() map[string]bool {
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

func Filter(word string) (string, bool) {
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

func CleanWords(ary []string, stopwords map[string]bool) []string {
	out := make([]string, 0, len(ary))
	set := make(map[string]bool, len(ary)/10)
	for _, v := range ary {
		if str, isWord := Filter(v); str != "" && isWord && !stopwords[str] && !set[str] {
			set[str] = true
			out = append(out, v) // Original word
		}
	}
	return out
}

func ParseStringToNode(contents string) []string {

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

func SearchBook(keyword string, stopwords map[string]bool, c_book, c_score *mgo.Collection, conn redis.Conn) (r []bson.M) {
	var m_ []bson.M
	var wg sync.WaitGroup

	// Search by TF-IDF score
	wg.Add(1)
	go func() {
		words := CleanWords(ParseStringToNode(keyword), stopwords)
		list := GetBooksByWords(words, c_score) // Get Book list by score
		m_ = append(m_, GetBookByList(list, c_book)...)
		wg.Done()
	}()

	// Search by Fuzzy search in author/title/original title field of DB
	wg.Add(1)
	go func() {
		m_ = append(m_, GetBookByList(GetBooksByKeywordRedis(keyword, conn), c_book)...)
		wg.Done()
	}()
	wg.Wait()

	return m_
}

func GetBooksByKeywordRedis(keyword string, conn redis.Conn) []int {
	keys, err := redis.Strings(conn.Do("Keys", "*"+keyword+"*"))
	if err != nil {
		panic(err)
	}
	list := make([]int, 0)
	for _, word := range keys {
		id, _ := redis.Int(conn.Do("GET", word))
		list = append(list, id)
	}
	return list
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

	for _, v := range m_ {
		v["txtlink"] = CreateTxtLink(v["booklink"].(string))
	}
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

	for _, v := range m__ {
		v["txtlink"] = CreateTxtLink(v["booklink"].(string))
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
	vs := NewValSorter(results)
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

func CreateTxtLink(bookUrl string) string {
	txtLink := ""
	reid := regexp.MustCompile(`files/(.+).html`)
	if array := reid.FindStringSubmatch(bookUrl); len(array) == 2 {
		txtLink = "txt/" + string(array[1]) + ".txt"
	}
	return txtLink
}

func GetTxtContents(filename string, c *mgo.Collection) (string, []string) {
	ary := make([]string, 0)
	if _, err := os.Stat(local_path + filename); err != nil {
		// Not found the txt file
		// Download it from official
		GenTxtFileByName(filename, c)
	}
	f, err := ioutil.ReadFile(local_path + filename)
	if err != nil {
		panic(err)
	}
	for _, v := range strings.Split(string(f), "\n") {
		ary = append(ary, v)
	}
	name, id, m := "", strings.Split(filename, "_")[0], bson.M{}
	if idInt, err := strconv.Atoi(id); err == nil {
		if err := c.Find(bson.M{"id": idInt}).One(&m); err == nil { // Do Query
			name = m["author"].(string) + " - " + m["title"].(string)
		}
	}
	return name, ary
}

func GenTxtFileByName(filename string, c *mgo.Collection) {
	url_path, m := "", bson.M{}
	id := strings.Split(filename, "_")[0]
	if idInt, err := strconv.Atoi(id); err == nil {
		if err := c.Find(bson.M{"id": idInt}).One(&m); err == nil { // Do Query
			url_path = m["booklink"].(string) // URL Path
			if err = os.Chdir(html_path); err != nil {
				panic(err)
			}
			if _, err = exec.Command("wget", url_path).Output(); err != nil {
				panic(err)
			}
		}
	}
	if err := os.Chdir(local_path); err != nil {
		panic(err)
	}
	// Use Iconv to do the conversion
	cd, err := iconv.Open("utf-8", "shift-jis") // convert shift-jis to utf-8
	if err != nil {
		return
	}
	defer cd.Close()
	htmlname := html_path + path.Base(url_path)
	txtname := local_path + filename
	GenTxt(htmlname, txtname, cd)
	if err := os.Chdir(work_folder); err != nil {
		panic(err)
	}
}

// Generate Txt file from original html file
func GenTxt(file, outpath string, cd iconv.Iconv) {
	re_pre := regexp.MustCompile("(?is)^.*<title>")
	re_en := regexp.MustCompile("(?i)<[^/!?r>]{2}[^>]*[/!?]?>")
	re_end1 := regexp.MustCompile("(?i)<[/!?][^r][^>]*>")
	re_end2 := regexp.MustCompile("(?i)<[!?b]r[^>]*>")
	re_post := regexp.MustCompile("(?s)(</div>\n)?<div class=\"bibliographical_information\">.*$")

	if f, err := ioutil.ReadFile(file); err == nil {
		str := re_pre.ReplaceAllString(string(f), "") // Section front of content
		str = re_post.ReplaceAllString(str, "")       // Section back of content
		str = re_en.ReplaceAllString(str, "")         // Signle tag
		str = re_end1.ReplaceAllString(str, "")       // Signle tag
		str = re_end2.ReplaceAllString(str, "")       // Signle tag

		// Do the conversion before write out
		// No additional encoding config for file needed
		if out, err := os.Create(outpath); err == nil {
			out.WriteString(cd.ConvString(str))
		}
	}
}
