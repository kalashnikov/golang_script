package main

import (
	"bytes"
	"fmt"
	//"bufio"
	//"github.com/microcosm-cc/bluemonday"
	//"github.com/russross/blackfriday"
	//"github.com/madari/goskirt"
	"github.com/kentaro/go-hoedown"
	"io/ioutil"
)

func main() {

	//input := bufio.NewScanner()
	data, _ := ioutil.ReadFile("1.md")

	//unsafe := blackfriday.MarkdownBasic(data)
	//html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)

	//skirt := goskirt.Goskirt{
	//	goskirt.EXT_AUTOLINK | goskirt.EXT_TABLES | goskirt.EXT_FENCED_CODE,
	//	goskirt.HTML_TOC | goskirt.HTML_SMARTYPANTS | goskirt.HTML_ESCAPE | goskirt.HTML_HARD_WRAP | goskirt.HTML_SAFELINK | goskirt.HTML_EXPAND_TABS,
	//}
	//skirt.WriteHTML(buf, data)

	parser := hoedown.NewHoedown(map[string]uint{
		"extensions":  hoedown.EXT_NO_INTRA_EMPHASIS | hoedown.EXT_AUTOLINK | hoedown.EXT_FENCED_CODE | hoedown.EXT_QUOTE,
		"renderModes": hoedown.HTML_USE_XHTML | hoedown.HTML_ESCAPE | hoedown.HTML_PRETTIFY,
	})

	buf := new(bytes.Buffer)
	parser.Markdown(buf, data)
	fmt.Println(buf.String())
}
