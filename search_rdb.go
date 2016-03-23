package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath" // Not included in path
	"strconv"
	"strings"
	"sync"
)

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

func Search(fname string, loy int, hiy int, wg *sync.WaitGroup, ch chan<- string) {
	defer wg.Done()

	file, err := os.Open(fname)
	if err != nil {
		panic(fname)
	}
	defer file.Close()

	r := bufio.NewReader(file)
	s, e := Readln(r) // Line 1
	s, e = Readln(r)  // Line 2
	s, e = Readln(r)  // Line 3
	s, e = Readln(r)  // Line 4
	s, e = Readln(r)  // Start from Line 5
	for e == nil {
		if !strings.Contains(s, "p") && strings.Contains(s, " ") {
			ary := strings.Split(s, " ")
			if y, err := strconv.Atoi(ary[1]); err == nil {
				if y >= loy && y <= hiy {
					ch <- fname
				}
			}
		}
		s, e = Readln(r)
	}
	ch <- ""
}

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: ./search_rdb.rb filter_string y_cordinate offset")
		return
	}

	keyword := "*" + string(os.Args[1]) + "*.rdb"

	ycord, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error: Argument 1 is not integer.")
		return
	}

	offset, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println("Error: Argument 2 is not integer.")
		return
	}

	//fmt.Printf("Parameter: %s %d %d => %d %d \n", keyword, ycord, offset, ycord-offset, ycord+offset)

	var fileList []string
	fileList, err = filepath.Glob(keyword)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	// Search each rdb file
	queue := make(chan string, 1)
	wg := &sync.WaitGroup{}
	for _, f := range fileList {
		wg.Add(1)
		go Search(f, ycord-offset, ycord+offset, wg, queue)
	}

	// Unique result
	// struct{} do not use memory
	set := make(map[string]struct{})
	go func() {
		defer wg.Done()
		for t := range queue {
			if t != "" {
				set[t] = struct{}{}
			}
		}
	}()
	wg.Wait()

	for f := range set {
		fmt.Println(f)
	}
}
