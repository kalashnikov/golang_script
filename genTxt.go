package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

//const folder = "html/"
//const out = "txt/"

func genTxt(file string,
	re_pre *regexp.Regexp,
	re_br *regexp.Regexp,
	re_en *regexp.Regexp,
	re_enn *regexp.Regexp,
	re_div *regexp.Regexp,
	re_strong *regexp.Regexp,
	re_ruby *regexp.Regexp,
	re_span *regexp.Regexp,
	re_h4 *regexp.Regexp,
	re_em *regexp.Regexp,
	re_post *regexp.Regexp) {

	outpath := "txt/" + strings.Split(path.Base(file), ".")[0] + ".txt"
	if f, err := ioutil.ReadFile(file); err == nil {
		str := re_pre.ReplaceAllString(string(f), "")
		str = re_br.ReplaceAllString(str, "")
		str = re_div.ReplaceAllString(str, "")
		str = re_strong.ReplaceAllString(str, "")
		str = re_ruby.ReplaceAllString(str, "")
		str = re_span.ReplaceAllString(str, "")
		str = re_h4.ReplaceAllString(str, "")
		str = re_em.ReplaceAllString(str, "")
		str = re_post.ReplaceAllString(str, "")
		str = re_en.ReplaceAllString(str, "")
		str = re_enn.ReplaceAllString(str, "")
		fmt.Println(outpath)
		if out, err := os.Create(outpath); err == nil {
			out.WriteString(str)
		}
	}
}

func main() {
	//file := folder + "1142_20453.html"
	//outpath := out + strings.Split(path.Base(file), ".")[0] + ".txt"

	// Regular expression for html clean-up
	re_pre := regexp.MustCompile("(?s)^.*^.*<div class=\"main_text\">")
	re_br := regexp.MustCompile("<br ?/?>")
	re_en := regexp.MustCompile("<[a-z]+.*/>")
	re_enn := regexp.MustCompile("</[a-z]+>")
	re_div := regexp.MustCompile("(?s)<div[^>]*>(.*?)</div>")
	re_strong := regexp.MustCompile("(?s)<strong[^>]*>(.*?)</strong>")
	re_ruby := regexp.MustCompile("(?s)<ruby[^>]*>(.*?)</ruby>")
	re_span := regexp.MustCompile("(?s)<span[^>]*>(.*?)</span>")
	re_h4 := regexp.MustCompile("(?s)<h4[^>]*>(.*?)</h4>")
	re_em := regexp.MustCompile("(?s)<em[^>]*>(.*?)</em>")
	re_post := regexp.MustCompile("(?s)(</div>\n)?<div class=\"bibliographical_information\">.*$")

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				genTxt(file, re_pre, re_br, re_en, re_enn, re_div, re_strong, re_ruby, re_span, re_h4, re_em, re_post)
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
