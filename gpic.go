package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
)

type Pic struct {
	Title            string
	Image            string
	Creator          string
	Artist_link      string
	Attribution      string
	Attribution_link string
	Source           string
	Link             string
}

const file_name = "imax.json"
const url_path = "https://www.gstatic.com/culturalinstitute/tabext/imax.json"
const gart = "https://www.google.com/culturalinstitute/"

func PrepareFile() {
	// Remove previous package
	if _, err := os.Stat(file_name); err == nil {
		err1 := os.Remove(file_name)
		if err1 != nil {
			fmt.Printf("Remove failed: %s\n", err1)
			panic(err1)
		}
	}

	// Download
	_, err := exec.Command("wget", url_path).Output()
	if err != nil {
		fmt.Printf("Download failed: %s\n", err)
		panic(err)
	}
}

func main() {
	PrepareFile()

	urlList := make([]string, 0)
	if f, err := ioutil.ReadFile(file_name); err == nil {
		var m []Pic
		if err := json.Unmarshal(f, &m); err == nil {
			for _, v := range m {
				//fmt.Printf("<a href=\" %s%s \"><div class=\"wrapper\">\n<img alt=\" %s \" src=\" %s \" srcset=\" %s=s1200-rw 2x\" />\n<div class=\"desc\"><p class=\"descc\"> %s - %s <BR/> %s </p></div>\n</div></a>\n ", gart, v.Link, v.Title, v.Image, v.Image, v.Creator, v.Title, v.Attribution)
				urlList = append(urlList, v.Image)
			}
		}
	}

	// Download pics
	if _, err := os.Stat("gpics"); err != nil {
		err = os.Mkdir("gpics", 0777)
		if err != nil {
			fmt.Printf("mkdir failed: %s\n", err)
			panic(err)
		}
	}

	err := os.Chdir("gpics")
	if err != nil {
		panic(err)
	}

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			for url := range tasks {
				GetFile(url)
			}
			wg.Done()
		}()
	}

	for _, u := range urlList {
		tasks <- u
	}
	close(tasks)
	wg.Wait()
}

func GetFile(url string) {
	u := url + "=s1200-rw"
	_, err := exec.Command("wget", u).Output()
	if err != nil {
		fmt.Printf("Download failed: %s | %s\n", err, u)
		panic(err)
	} else {
		fmt.Printf("Download finished. | %s\n", u)
	}
}
