package main

//
// Read-in keywords and scan each document to calculate TF(term frequency)
//
//

import (
	"encoding/csv"
	"fmt"
	"github.com/bluele/mecab-golang"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

func calTF(file string, idfList map[string]float64, tfList map[string]map[string]float64) {
	results := map[string]float64{}
	if f, err := ioutil.ReadFile(file); err == nil {
		for _, v := range parseToNode(string(f)) {
			if v == "" {
				continue
			}
			results[v]++
		}
	}
	file_id := ""
	if strings.Contains(file, "_") {
		file_id = strings.Split(path.Base(file), "_")[0]
	} else {
		file_id = strings.Split(path.Base(file), ".")[0]
	}
	tfList[file_id] = results
}

func main() {
	start := time.Now()
	cpunum := runtime.NumCPU()

	tfList := make(map[string]map[string]float64, 20000)
	idfList := make(map[string]float64, 220000)

	// Open File
	f, ferr := os.Open("result_0623.csv")
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}
	defer f.Close()

	// Init idfList
	reader := csv.NewReader(f)
	for {
		ary, rerr := reader.Read()
		if rerr == io.EOF {
			break
		} else if rerr != nil {
			fmt.Printf("CSV Reader failed: %s\n", rerr)
			panic(rerr)
		}
		if n, err := strconv.ParseFloat(ary[2], 64); err == nil {
			idfList[ary[0]] = n
		}
	}

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				calTF(file, idfList, tfList)
			}
			wg.Done()
		}()
	}

	// Collect frequency for each file based on each word
	if a, err := filepath.Glob("txt/*.txt"); err == nil {
		for _, file := range a {
			tasks <- file
		}
	}
	close(tasks)

	wg.Wait()

	// Create the final data strcture
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

	// Check the result
	fmt.Println("#### Result ####")
	for word, m := range tfFinal {
		fmt.Printf("%s [%g]: ", word, idfList[word])
		for doc, v := range m {
			fmt.Printf("%s(%g) ", doc, v)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Time used: %v", time.Since(start))
}
