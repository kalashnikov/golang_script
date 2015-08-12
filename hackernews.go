package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sync"
)

type News struct {
	Id    int
	Title string
	Url   string
	Score int
}

func GetBody(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	h, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return h
}

func Fill(url string, data *News, wg *sync.WaitGroup) {
	i := GetBody(url)
	json.Unmarshal(i, data)
	wg.Done()
}

const work_folder = "/var/opt/www/go/"

func main() {
	err := os.Chdir(work_folder)
	if err != nil {
		panic(err)
	}

	url := "https://hacker-news.firebaseio.com/v0/topstories.json?print=pretty"
	h := GetBody(url)
	var array []int
	json.Unmarshal(h, &array)

	data := make([]News, 40)

	var wg sync.WaitGroup
	for idx, i := range array[0:39] {
		wg.Add(1)
		iurl := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json?print=pretty", i)
		go Fill(iurl, &data[idx], &wg)
	}
	wg.Wait()

	// Download and create new output file
	outputFile := "hackernews.csv"
	if out, err := os.Create(outputFile); err == nil {
		for _, v := range data {
			if v.Title == "" {
				continue
			}

			turl := v.Url
			if turl == "" {
				turl = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", v.Id)
			}
			re := regexp.MustCompile("[,\"]")
			str := fmt.Sprintf("%d,\"%s\",\"%s\"\n", v.Score, re.ReplaceAllString(v.Title, " "), turl)
			out.WriteString(str)
		}
	}
}
