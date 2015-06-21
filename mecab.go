package main

import (
	"fmt"
	"github.com/bluele/mecab-golang"
	"golang.org/x/text/width"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// https://github.com/dbravender/go_mapreduce/blob/master/src/mapreduce/mapreduce.go
func MapReduce(mapper func(interface{}, chan interface{}),
	reducer func(chan interface{}, chan interface{}),
	input chan interface{},
	pool_size int) interface{} {
	reduce_input := make(chan interface{})
	reduce_output := make(chan interface{})
	worker_output := make(chan chan interface{}, pool_size)
	go reducer(reduce_input, reduce_output)
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
	return <-reduce_output
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

func reducer(input chan interface{}, output chan interface{}) {
	results := map[string]int{}
	for new_matches := range input {
		for key, value := range new_matches.(map[string]int) {
			previous_count, exists := results[key]
			if !exists {
				results[key] = value
			} else {
				results[key] = previous_count + value
			}
		}
	}
	output <- results
}

func getFiles(folder string) chan interface{} {
	output := make(chan interface{})
	go func() {
		if a, err := filepath.Glob(folder + "/*.txt"); err == nil {
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

func main() {
	start := time.Now()

	// Generate candidate words
	result := MapReduce(wordcount, reducer, getFiles("txt"), 20).(map[string]int)

	stopwords := getStopWords()

	// Check Result
	var keys []string
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	words := cleanWords(keys, stopwords)

	for _, v := range words {
		fmt.Printf("%s: %d\n", v, result[v])
	}
	fmt.Printf("Time used: %v", time.Since(start))
}
