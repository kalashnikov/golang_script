package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath" // Not included in path
	"runtime"
	"strings"
	"sync"
)

var wg sync.WaitGroup

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

func Collect(fname string, ch chan<- string) {
	file, err := os.Open(fname)
	if err != nil {
		panic(fname)
	}
	defer file.Close()

	buf := bytes.NewBufferString("")

	r := bufio.NewReader(file)
	s, e := Readln(r) // Line 1
	s, e = Readln(r)  // Line 2
	s, e = Readln(r)  // Line 3
	s, e = Readln(r)  // Line 4
	s, e = Readln(r)  // Start from Line 5
	for e == nil {
		if strings.Contains(s, "p") {
			if buf.Len() > 0 {
				ch <- buf.String()
			}
			buf = bytes.NewBufferString("")
		} else if strings.Contains(s, " ") {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(s)
		}
		s, e = Readln(r)
	}
	ch <- ""
}

func main() {

	keyword := "*.rdb"
	outpath := "merge_out.rdb"
	if len(os.Args) > 1 {
		keyword = "*" + string(os.Args[1]) + keyword
		outpath = string(os.Args[1]) + "-" + outpath
	}

	cpunum := runtime.NumCPU()
	tasks := make(chan string, 16)
	queue := make(chan string, 16)

	// Collecting shapes
	// Unique result: struct{} do not use memory
	set := make(map[string]struct{})
	go func() {
		for t := range queue {
			if t != "" {
				set[t] = struct{}{}
			}
		}
	}()

	// Get shapes
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for f := range tasks {
				Collect(f, queue)
			}
			wg.Done()
		}()
	}

	// RDB File Lists
	if a, err := filepath.Glob(keyword); err == nil {
		for _, f := range a {
			if f == outpath {
				continue
			}
			tasks <- string(f)
		}
	}
	close(tasks)
	wg.Wait()

	// Remove previous package
	if _, err := os.Stat(outpath); err == nil {
		err := os.Remove(outpath)
		if err != nil {
			fmt.Printf("### Remove failed: %s\n", err)
			panic(err)
		} else {
			fmt.Println("### Remove file finished.")
		}
	}

	// Write the output RDB file
	tmpFile, _ := filepath.Glob(keyword)
	if out, err := os.Create(outpath); err == nil {
		poly_cnt := len(set)

		if file, err := os.Open(tmpFile[0]); err == nil {
			r := bufio.NewReader(file)
			s, _ := Readln(r) // Header line 1
			out.WriteString(s + "\n")

			s, _ = Readln(r) // Header line 2
			out.WriteString(s + "\n")

			s, _ = Readln(r) // Header line 3
			ary := strings.Split(s, " ")
			line := fmt.Sprintf("%d %d 0 ", poly_cnt, poly_cnt) + strings.Join(ary[3:], " ") + "\n"
			out.WriteString(line)
		} else {
			panic(tmpFile[0])
		}

		cnt := 1
		for f := range set {
			out.WriteString(fmt.Sprintf("p %d %d\n", cnt, strings.Count(f, "\n")))
			//out.WriteString(fmt.Sprintf("p %d %d\n", cnt, poly_cnt))
			out.WriteString(f)
			out.WriteString("\n")
			cnt += 1
		}
	}
	fmt.Printf("### Done. Output file: %s.\n", outpath)
}
