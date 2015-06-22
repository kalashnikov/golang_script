package main

import (
	"fmt"
	"github.com/bluele/mecab-golang"
	"golang.org/x/text/width"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// https://github.com/dbravender/go_mapreduce/blob/master/src/mapreduce/mapreduce.go
func MapReduce(mapper func(interface{}, chan interface{}),
	reducer func(chan interface{}, chan interface{}, chan interface{}),
	input chan interface{},
	pool_size int) (interface{}, interface{}) {
	reduce_input := make(chan interface{})
	reduce_output := make(chan interface{})
	reduce_output2 := make(chan interface{})
	worker_output := make(chan chan interface{}, pool_size)
	go reducer(reduce_input, reduce_output, reduce_output2)
	go func() {
		for worker_chan := range worker_output {
			reduce_input <- <-worker_chan
		}
		close(reduce_input)
	}()
	go func() {
		for item := range input {
			my_chan := make(chan interface{})
			go mapper(item, my_chan)
			worker_output <- my_chan
		}
		close(worker_output)
	}()
	return <-reduce_output, <-reduce_output2
}

// https://github.com/dbravender/go_mapreduce/blob/master/src/wordcount/wordcount.go
func wordcount(filename interface{}, output chan interface{}) {
	results := map[string]int{}
	if f, err := ioutil.ReadFile(filename.(string)); err == nil {
		for _, v := range parseToNode(string(f)) {
			if v == "" {
				continue
			}
			results[v]++
		}
	}
	output <- results
}

func reducer(input chan interface{}, output chan interface{}, output2 chan interface{}) {
	results := map[string]int{}
	results2 := map[string]int{}
	for new_matches := range input {
		for key, value := range new_matches.(map[string]int) {
			previous_count, exists := results[key]
			if !exists {
				results[key] = value
				results2[key] = 1
			} else {
				results[key] = previous_count + value
				results2[key] = previous_count + 1
			}
		}
	}
	output <- results
	output2 <- results2
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

func calculateTFIDF(tf interface{}, idf interface{}, folder string) (map[string]float64, map[string]float64) {

	tf_ := tf.(map[string]int)
	idf_ := idf.(map[string]int)

	tf_out := make(map[string]float64, len(tf_))
	idf_out := make(map[string]float64, len(idf_))

	// Get Total term/word count
	tc := 0
	for _, v := range tf_ {
		tc += v
	}

	// Get Total document count
	dc := 0
	if a, err := filepath.Glob("txt/*.txt"); err == nil {
		dc = len(a)
	}

	for k, v := range tf_ {
		tf_out[k] = float64(v)
	}

	// Inverse Document Frequency = Log ( document count / Total document count )
	for k, v := range idf_ {
		idf_out[k] = math.Log(float64(dc) / float64(v))
	}

	return tf_out, idf_out
}

func main() {
	start := time.Now()

	// Generate candidate words
	tf_, idf_ := MapReduce(wordcount, reducer, getFiles("txt/*.txt"), 20)
	tf, idf := calculateTFIDF(tf_, idf_, "txt/*.txt")

	stopwords := getStopWords()

	// Check Result
	var keys []string
	for k := range tf {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	words := cleanWords(keys, stopwords)

	for _, v := range words {
		fmt.Printf("%s,%g,%g\n", v, tf[v], idf[v])
	}
	fmt.Printf("Time used: %v", time.Since(start))
}
