package main

import (
	"fmt"
	"github.com/kalashnikov/golang_script/obm"
	"github.com/kalashnikov/golang_script/utility"
	"gopkg.in/mgo.v2"
	"regexp"
	"sync"
)

func GetIDs(urlStr string, wg *sync.WaitGroup, urls *[]string) {
	defer wg.Done()

	// Use Cookie (Broken - no difference)
	contents := geturl.GetDocByURL(urlStr)

	// Check data exist
	desc := geturl.GetSingleText(contents, "div.mdMN02Desc")
	if desc == "" {
		return
	}

	reg := regexp.MustCompile("product/(.+)/zh.Hant")
	tmpurl := geturl.GetAttrs(contents, "li.mdMN02Li a", "href")
	for _, v := range tmpurl {
		if v == "" {
			continue
		}
		str := reg.FindStringSubmatch(v)[1]
		*urls = append(*urls, str)
	}
}

func main() {
	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c_stickers := session.DB("obmWeb").C("stickers")
	c_themes := session.DB("obmWeb").C("themes")

	originalStr := "https://store.line.me/stickershop/showcase/new/zh-Hant?page="
	creatorStr := "https://store.line.me/stickershop/showcase/top_creators/zh-Hant?page="
	themeStr := "https://store.line.me/themeshop/showcase/zh-Hant?page="

	// Get the new id list
	urls := make([]string, 0)
	var wg sync.WaitGroup
	for _, v := range []string{originalStr, creatorStr, themeStr} {
		for i := 0; i < 10; i++ {
			urlStr := fmt.Sprintf("%s%d", v, i+1)
			wg.Add(1)
			go GetIDs(urlStr, &wg, &urls)
		}
	}
	wg.Wait()

	// Get the data
	for _, v := range urls {
		wg.Add(1)
		go obm.GetStickersData(v, c_stickers, c_themes, &wg)
	}
	wg.Wait()
}
