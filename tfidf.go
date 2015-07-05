package main

//
// Read-in keywords and scan each document to calculate TF(term frequency)
//
//

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/bluele/mecab-golang"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
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

func calTF(file string,
	idfMap map[string]float64,
	tfMap map[string]map[string]float64,
	titleMap map[string]string) {

	total_words := 0.0
	results := map[string]float64{}

	// Get File ID
	file_id := ""
	if strings.Contains(file, "_") {
		file_id = strings.Split(path.Base(file), "_")[0]
	} else {
		file_id = strings.Split(path.Base(file), ".")[0]
	}

	for _, v := range parseToNode(titleMap[file_id]) {
		results[v] += 5 // one for ten
	}

	// Calculate word count
	if f, err := ioutil.ReadFile(file); err == nil {
		for _, v := range parseToNode(string(f)) {
			if v == "" {
				continue
			}
			results[v]++
			total_words++
		}
	}

	// Calculate TF
	for _, v := range results {
		v = v / total_words
	}

	tfMap[file_id] = results
}

func LoadIDFMap(file string) map[string]float64 {
	// Open File
	f, ferr := os.Open(file)
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}
	defer f.Close()

	idfMap := make(map[string]float64, 220000)

	// Init idfMap
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
			idfMap[ary[0]] = n
		}
	}
	return idfMap
}

func LoadTitleAuthorMap(file string) map[string]string {
	// Open File
	f, ferr := os.Open(file)
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}
	defer f.Close()

	titleMap := make(map[string]string, 15000)

	// Init idfMap
	reader := csv.NewReader(f)
	for {
		ary, rerr := reader.Read()
		if rerr == io.EOF {
			break
		} else if rerr != nil {
			fmt.Printf("CSV Reader failed: %s\n", rerr)
			panic(rerr)
		}

		titleMap[ary[0]] = ary[1] + " " + ary[2]
	}
	return titleMap
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	//start := time.Now()
	cpunum := runtime.NumCPU()

	tfMap := make(map[string]map[string]float64, 20000)
	idfMap := LoadIDFMap("result_0623.csv")
	titleMap := LoadTitleAuthorMap("book.csv")

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				calTF(file, idfMap, tfMap, titleMap)
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
	tfFinal := make(map[string]map[string]float64, len(tfMap))
	for doc, m := range tfMap {
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
	word := ""
	for w, m := range tfFinal {
		word = strings.Trim(w, " ")
		if word == "" || strings.Contains(word, ",") {
			continue
		}
		if idf, ok := idfMap[word]; ok {
			fmt.Printf("%s,%g,", word, idf)
			list := make([]string, 0, len(m))
			for doc, v := range m {
				list = append(list, fmt.Sprintf("%s_%g", doc, v))
			}
			fmt.Printf("%s\n", strings.Join(list, ","))
		}
	}

	//fmt.Printf("Time used: %v", time.Since(start))
}
