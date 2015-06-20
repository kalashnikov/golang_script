package main

//
// Transform Aozora HTML file into txt
//

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

type RegPack struct {
	re_pre    *regexp.Regexp
	re_en     *regexp.Regexp
	re_div    *regexp.Regexp
	re_strong *regexp.Regexp
	re_ruby   *regexp.Regexp
	re_span   *regexp.Regexp
	re_h4     *regexp.Regexp
	re_em     *regexp.Regexp
	re_post   *regexp.Regexp
}

func genTxt(file string, reg RegPack) {
	outpath := "txt/" + strings.Split(path.Base(file), ".")[0] + ".txt"
	if f, err := ioutil.ReadFile(file); err == nil {
		str := reg.re_pre.ReplaceAllString(string(f), "") // Section front of content
		str = reg.re_post.ReplaceAllString(str, "")       // Section back of content
		str = reg.re_div.ReplaceAllString(str, "")
		str = reg.re_strong.ReplaceAllString(str, "")
		str = reg.re_ruby.ReplaceAllString(str, "")
		str = reg.re_span.ReplaceAllString(str, "")
		str = reg.re_h4.ReplaceAllString(str, "")
		str = reg.re_em.ReplaceAllString(str, "")
		str = reg.re_en.ReplaceAllString(str, "") // Signle tag
		fmt.Println(outpath)
		if out, err := os.Create(outpath); err == nil {
			out.WriteString(str)
		}
	}
}

func main() {

	cpunum := runtime.NumCPU()

	// Regular expression for html clean-up
	var reg RegPack
	reg.re_pre = regexp.MustCompile("(?s)^.*^.*<div class=\"main_text\">")
	reg.re_en = regexp.MustCompile("<(?i)[/|!]?[a-z]+.*/?>")
	reg.re_div = regexp.MustCompile("(?is)<div[^>]*>(.*?)</div>")
	reg.re_strong = regexp.MustCompile("(?is)<strong[^>]*>(.*?)</strong>")
	reg.re_ruby = regexp.MustCompile("(?is)<ruby[^>]*>(.*?)</ruby>")
	reg.re_span = regexp.MustCompile("(?is)<span[^>]*>(.*?)</span>")
	reg.re_h4 = regexp.MustCompile("(?is)<h4[^>]*>(.*?)</h4>")
	reg.re_em = regexp.MustCompile("(?is)<em[^>]*>(.*?)</em>")
	reg.re_post = regexp.MustCompile("(?s)(</div>\n)?<div class=\"bibliographical_information\">.*$")

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				genTxt(file, reg)
			}
			wg.Done()
		}()
	}

	if a, err := filepath.Glob("html/*.html"); err == nil {
		for _, file := range a {
			tasks <- file
		}
	}
	close(tasks)

	wg.Wait()
}
