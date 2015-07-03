package main

//
// Transform Aozora HTML file into txt
//

import (
	"fmt"
	"github.com/qiniu/iconv"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type RegPack struct {
	re_pre  *regexp.Regexp
	re_en   *regexp.Regexp
	re_end1 *regexp.Regexp
	re_end2 *regexp.Regexp
	re_post *regexp.Regexp
}

func genTxt(file string, reg RegPack, cd iconv.Iconv) {
	outpath := "txtraw/" + strings.Split(path.Base(file), ".")[0] + ".txt"
	if f, err := ioutil.ReadFile(file); err == nil {
		str := reg.re_pre.ReplaceAllString(string(f), "") // Section front of content
		str = reg.re_post.ReplaceAllString(str, "")       // Section back of content
		str = reg.re_en.ReplaceAllString(str, "")         // Signle tag
		str = reg.re_end1.ReplaceAllString(str, "")       // Signle tag
		str = reg.re_end2.ReplaceAllString(str, "")       // Signle tag
		fmt.Println(outpath)

		// Do the conversion before write out
		// No additional encoding config for file needed
		if out, err := os.Create(outpath); err == nil {
			out.WriteString(cd.ConvString(str))
		}
	}
}

func main() {

	start := time.Now()
	cpunum := runtime.NumCPU()

	// Regular expression for html clean-up
	var reg RegPack
	reg.re_pre = regexp.MustCompile("(?is)^.*<title>")
	reg.re_en = regexp.MustCompile("(?i)<[^/!?r>]{2}[^>]*[/!?]?>")
	reg.re_end1 = regexp.MustCompile("(?i)<[/!?][^r][^>]*>")
	reg.re_end2 = regexp.MustCompile("(?i)<[!?b]r[^>]*>")
	reg.re_post = regexp.MustCompile("(?s)(</div>\n)?<div class=\"bibliographical_information\">.*$")

	// Use Iconv to do the conversion
	cd, err := iconv.Open("utf-8", "shift-jis") // convert shift-jis to utf-8
	if err != nil {
		fmt.Println("iconv.Open failed!")
		return
	}
	defer cd.Close()

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				genTxt(file, reg, cd)
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
	fmt.Printf("Time used: %v", time.Since(start))
}
