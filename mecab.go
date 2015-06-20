package main

import (
	"fmt"
	"github.com/bluele/mecab-golang"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
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

func main() {
	result := MapReduce(wordcount, reducer, getFiles("txt"), 20).(map[string]int)

	// Check Result
	var keys []string
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, v := range keys {
		fmt.Printf("%s: %d\n", v, result[v])
	}
}
