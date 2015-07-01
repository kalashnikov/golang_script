package geturl

import (
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/css"
	html "github.com/moovweb/gokogiri/html"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

func GetDocByURL(url string) *html.HtmlDocument {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	h, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	doc, err := gokogiri.ParseHtml(h)
	if err != nil {
		panic(err)
	}

	return doc
}

func GetDocByURLWithCookie(url string, gCookieJar *cookiejar.Jar) *html.HtmlDocument {
	httpclient := http.Client{
		CheckRedirect: nil,
		Jar:           gCookieJar,
	}

	resp, err := httpclient.Get(url)
	if err != nil {
		panic(err)
	}

	h, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	doc, err := gokogiri.ParseHtml(h)
	if err != nil {
		panic(err)
	}

	return doc
}

func TestGet(url string, gCookieJar *cookiejar.Jar) []*http.Cookie {
	httpclient := http.Client{
		CheckRedirect: nil,
		Jar:           gCookieJar,
	}

	resp, _ := httpclient.Get(url)
	defer resp.Body.Close()

	cookies := resp.Cookies()

	c1 := &http.Cookie{
		Name:   "_ga",
		Value:  "GA1.2.1774881701.1411111177",
		Path:   "/",
		Domain: ".line.me",
	}
	c2 := &http.Cookie{
		Name:   "store_lang",
		Value:  "zh-hant",
		Path:   "/",
		Domain: ".line.me",
	}
	c3 := &http.Cookie{
		Name:   "store_locale",
		Value:  "zh_TW",
		Path:   "/",
		Domain: ".line.me",
	}
	cookies = append(cookies, c1, c2, c3)
	return cookies
}

func GetSingleText(body *html.HtmlDocument, cssStr string) string {
	nodeArr, err := body.Search(css.Convert(cssStr, css.GLOBAL))
	if err != nil {
		panic(err)
	}
	if len(nodeArr) == 0 {
		return ""
	}
	return strings.TrimSpace(nodeArr[0].FirstChild().String())
}

func GetFirstAttr(body *html.HtmlDocument, cssStr, attr string) string {
	nodeArr, err := body.Search(css.Convert(cssStr, css.GLOBAL))
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(nodeArr[0].Attr(attr))
}

func GetAttrs(body *html.HtmlDocument, cssStr, attr string) []string {
	nodeArr, err := body.Search(css.Convert(cssStr, css.GLOBAL))
	if err != nil {
		panic(err)
	}
	ary := make([]string, len(nodeArr))
	for _, v := range nodeArr {
		ary = append(ary, strings.TrimSpace(v.Attr(attr)))
	}
	return ary
}
