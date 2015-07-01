package main

//
// Website Server using Golang
// By Kala.Kuo http://kalakuo.info
//

import (
	"encoding/csv"
	"github.com/Shaked/gomobiledetect"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"github.com/kalashnikov/golang_script/book"
	"github.com/kalashnikov/golang_script/note"
	"github.com/kalashnikov/golang_script/obm"
	"github.com/martini-contrib/auth"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	//"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

type ResultArray []bson.M

type TemplateBag struct {
	Title string
	Msg   string
	Ary   ResultArray
	Ary2  ResultArray
	Ary3  ResultArray
	List  []string
}

func main() {

	rand.Seed(time.Now().UTC().UnixNano())

	// Stop word list
	stopwords := book.GetStopWords()

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c_book := session.DB("aozora").C("books_go")
	c_score := session.DB("aozora").C("tf_idf")
	c_stickers := session.DB("obmWeb").C("stickers")
	c_themes := session.DB("obmWeb").C("themes")

	m := martini.Classic()

	// render html templates from directory
	m.Use(render.Renderer())

	// Home
	m.Get("/", func(r render.Render) {
		ary := []TemplateBag{}
		p1 := TemplateBag{Title: "Kala", Msg: "Google"}
		p2 := TemplateBag{Title: "Ashley", Msg: "Tencent"}
		p3 := TemplateBag{Title: "Mama", Msg: "Kaohsiung"}
		ary = append(ary, p1, p2, p3)
		r.HTML(200, "index", ary)
	})

	// ------------------------------------------------------------------------------------- //
	// -------------------------------       OBM SHOP       -------------------------------- //
	// ------------------------------------------------------------------------------------- //

	// Official Sticker
	m.Get("/lines/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_stickers, 0, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Original Sticker
	m.Get("/clines/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_stickers, 1, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 25
	m.Get("/dollar25/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_stickers, 2, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 50
	m.Get("/dollar50/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_stickers, 3, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 75
	m.Get("/dollar75/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_stickers, 4, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Themes
	m.Get("/themes/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.GenStickerBag(detect, c_themes, 5, keyword, tag)
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Detail sticker page
	m.Get("/detail/:id", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		bag := obm.GetStickersDetail(params["id"], c_stickers, c_themes)
		re.HTML(200, "darkly", bag)
	})

	// ------------------------------------------------------------------------------------- //

	m.Get("/book/", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		if _, err := os.Stat("/var/opt/www/go/ranklist.md"); err == nil {
			if b, err := ioutil.ReadFile("ranklist.md"); err == nil {
				re.HTML(200, "rank", string(b))
			}
		} else {
			url := "/book/random"
			http.Redirect(w, r, url, 302)
		}
	})

	m.Get("/book/:str", func(params martini.Params, r render.Render) {
		keyword := params["str"]
		if keyword == "random" {
			if f, ferr := os.Open("authorList.csv"); ferr != nil {
				panic(ferr)
			} else {
				// Read first line only
				reader := csv.NewReader(f)
				if ary, rerr := reader.Read(); rerr == nil {
					keyword = ary[rand.Int()%len(ary)]
				}
			}
		}
		m_ := book.GetBooksByKeyword(keyword, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		r.HTML(200, "book", bag)
	})

	m.Get("/search-book/", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		keyword := ""
		if f, ferr := os.Open("authorList.csv"); ferr != nil {
			panic(ferr)
		} else {
			// Read first line only
			reader := csv.NewReader(f)
			if ary, rerr := reader.Read(); rerr == nil {
				keyword = ary[rand.Int()%len(ary)]
			}
		}
		m_ := book.GetBooksByKeyword(keyword, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "search", bag)
	})

	m.Get("/search-book/:str", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		keyword := params["str"]
		words := book.CleanWords(book.ParseStringToNode(keyword), stopwords)
		list := book.GetBooksByWords(words, c_score) // Get Book list by score
		m_ := book.GetBookByList(list, c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "search", bag)
	})

	m.Post("/search", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Post("/search-book", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/search-book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	// ------------------------------------------------------------------------------------- //

	m.Get("/note/", auth.BasicFunc(note.NoteAuth), func(w http.ResponseWriter, re render.Render) {
		msg := note.UpdateMenuFile()
		bag := TemplateBag{Title: "Note Contents List", Msg: msg}
		re.HTML(200, "md", bag)
	})

	m.Get("/note/:folder/:file", auth.BasicFunc(note.NoteAuth), func(params martini.Params, re render.Render) {
		name, msg := note.GetNoteContents(params["folder"] + "/" + params["file"])
		bag := TemplateBag{Title: name, Msg: msg}
		re.HTML(200, "md", bag)
	})

	// ------------------------------------------------------------------------------------- //

	m.Run()
}
