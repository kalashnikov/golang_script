package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath" // Not included in path
	"strings"
	"sync"
)

func check(path_slice []string, keyword string, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, path := range path_slice {
		if path == "" {
			return
		}

		contents := path
		if b, err := ioutil.ReadFile(path + "/rules"); err == nil {
			contents = contents + string(b)
		}
		if b, err := ioutil.ReadFile(path + "/rules.fill"); err == nil {
			contents = contents + string(b)
		}
		if b, err := ioutil.ReadFile(path + "/rules.svrf"); err == nil {
			contents = contents + string(b)
		}

		contents = strings.ToUpper(contents)
		if keyword == "" {
			fmt.Printf("%s | UNBOUNDED:%t, FIELD:%t, REPEAT 1:%t, MERGESHAPE:%t, LONGSHAPE:%t \n",
				path,
				strings.Contains(contents, "UNBOUNDED"),
				strings.Contains(contents, "FIELD"),
				strings.Contains(contents, "REPEAT 1"),
				strings.Contains(contents, "MERGESHAPE"),
				strings.Contains(contents, "LONGSHAPE"),
			)
		} else if strings.Contains(contents, keyword) {
			fmt.Println(path)
		}
	}
}

func main() {

	keyword := ""
	if len(os.Args) > 1 {
		keyword = string(os.Args[1])
	}

	folder_list := make(map[string][]string)

	a, err := filepath.Glob("/net/mosa/data/test_suite/dfm/fill2/fill/**/*")
	if err == nil {
		for _, p := range a {
			if _, err := os.Stat(p + "/rules"); err == nil && p != "" {
				folder_list[path.Dir(p)] = append(folder_list[path.Dir(p)], p)
			}
		}
	}

	wg := &sync.WaitGroup{}
	for _, path := range folder_list {
		wg.Add(1)
		go check(path, keyword, wg)
	}
	wg.Wait()
}
