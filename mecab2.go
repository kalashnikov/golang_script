package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/bluele/mecab-golang"
	"golang.org/x/text/width"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

var mutex = &sync.Mutex{}

func calTF(fname string,
	idfList map[string]float64,
	tfList map[string]map[string]float64,
	fnameList map[int]string,
	stopwords map[string]bool) {

	results := map[string]float64{}

	file, err := os.Open(fname)
	if err != nil {
		panic(fname)
	}
	defer file.Close()

	// Load the contents and pass into tokenizer
	// ReadFile is faster but memory consuming
	// Readln (read line-by-line) will be 10x slower
	if f, err := ioutil.ReadFile(fname); err == nil {
		for _, v := range parseToNode(string(f)) {
			if v == "" {
				continue
			} else if str, isWord := filter(v); str != "" && isWord && stopwords[str] {
				continue
			}
			results[v]++
		}
	}

	// Get author and title
	file_id, author, title := "", "", ""
	if strings.Contains(fname, "book_txt") {
		author, title = GetTitle(path.Base(fname))
		id := len(fnameList) + 1000001
		file_id = strconv.Itoa(id)
	} else {
		r := bufio.NewReader(file)
		s, _ := Readln(r)
		if strings.Contains(s, " ") {
			ary := strings.Split(s, " ")
			author, title = ary[0], ary[1]
		} else {
			title = s
		}

		// File id for aozora
		if strings.Contains(fname, "_") {
			file_id = strings.Split(path.Base(fname), "_")[0]
		} else {
			file_id = strings.Split(path.Base(fname), ".")[0]
		}
	}
	fmt.Printf("# Fin: %s - %s | %s\n", author, title, fname)

	// Add weighting to author & title
	for _, v := range parseToNode(author + " " + title) {
		if v == "" {
			continue
		} else if str, isWord := filter(v); str != "" && isWord && !stopwords[str] {
			continue
		}
		results[v] += 10
	}

	tfList[file_id] = results

	// Add into IDF list
	for v := range results {
		mutex.Lock()
		if _, ok := idfList[v]; ok {
			idfList[v] += 1.0
		} else {
			idfList[v] = 1.0
		}
		id, _ := strconv.Atoi(file_id)
		fnameList[id] = fname
		mutex.Unlock()
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	start := time.Now()
	cpunum := runtime.NumCPU()

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	searchStr := "*txt/*.txt"
	//searchStr := "book_txt/*.txt"

	// Filter out symbol and stopwords
	stopwords := getStopWords()

	tfList := make(map[string]map[string]float64, 20000)
	idfList := make(map[string]float64, 220000)
	fnameList := make(map[int]string, 20000)

	// spawn four worker goroutines
	fmt.Println("### calTF ###")
	tasks := make(chan string, 64)
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				calTF(file, idfList, tfList, fnameList, stopwords)
			}
			wg.Done()
		}()
	}

	if a, err := filepath.Glob(searchStr); err == nil {
		for _, f := range a {
			if f == "" {
				continue
			}
			tasks <- string(f)
		}
	}
	close(tasks)
	wg.Wait()

	fmt.Println("### Refine IDF ###")
	if a, err := filepath.Glob(searchStr); err == nil {
		total_doc := float64(len(a))
		for _, v := range idfList {
			v /= total_doc
		}
	}

	// Create the final data strcture
	fmt.Println("### Create final structure ###")
	tfFinal := make(map[string]map[string]float64, len(tfList))
	for doc, m := range tfList {
		for word, v := range m {
			if _, ok := tfFinal[word]; ok {
				tfFinal[word][doc] = v
			} else {
				_map := make(map[string]float64)
				tfFinal[word] = _map
				tfFinal[word][doc] = v
			}
		}
	}

	fmt.Println("### Write tf list ###")
	word := ""
	outputFile1 := "tflist.csv"
	if out, err := os.Create(outputFile1); err == nil {
		for w, m := range tfFinal {
			word = strings.Trim(w, " ")
			if word == "" || strings.Contains(word, ",") {
				continue
			}
			if idf, ok := idfList[word]; ok {
				str := fmt.Sprintf("%s,%g,", word, idf)
				list := make([]string, 0, len(m))
				for doc, v := range m {
					list = append(list, fmt.Sprintf("%s_%g", doc, v))
				}
				fstr := fmt.Sprintf("%s%s\n", str, strings.Join(list, ","))
				out.WriteString(fstr)
			} else {
				fmt.Printf("### ERROR - idfList not found: %s\n", word)
			}
		}
	}

	fmt.Println("### Write idf list ###")
	outputFile2 := "idflist.csv"
	if out, err := os.Create(outputFile2); err == nil {
		for w, v := range idfList {
			str := fmt.Sprintf("%s,%g\n", w, v)
			out.WriteString(str)
		}
	}

	fmt.Println("### Write fname list ###")
	outputFile3 := "fnameList.csv"
	if out, err := os.Create(outputFile3); err == nil {
		for w, v := range fnameList {
			str := fmt.Sprintf("%d,%s\n", w, v)
			out.WriteString(str)
		}
	}

	fmt.Printf("Time used: %v\n", time.Since(start))
}

func getFiles(folder string) chan interface{} {
	output := make(chan interface{})
	go func() {
		if a, err := filepath.Glob(folder); err == nil {
			for _, file := range a {
				output <- file
			}
		}
		close(output)
	}()
	return output
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
