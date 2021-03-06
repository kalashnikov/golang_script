package main

//
// Website Server using Golang
// By Kala.Kuo http://kalakuo.info
//

import (
	"encoding/csv"
	"github.com/Shaked/gomobiledetect"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/golang/glog"
	"github.com/kalashnikov/golang_script/book"
	"github.com/kalashnikov/golang_script/hack"
	"github.com/kalashnikov/golang_script/note"
	"github.com/kalashnikov/golang_script/obm"
	"github.com/kalashnikov/martini"
	"github.com/martini-contrib/auth"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"os"
	"time"
)

type ResultArray []bson.M

type TemplateBag struct {
	Title string
	Title_link string
	Msg   string
	Ary   ResultArray
	Ary2  ResultArray
	Ary3  ResultArray
	Data  []hack.News 
	Rank  []book.Rank
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
	
	// Connect to Redis : author/title/otitle to book id
	redisPool := book.InitRedisPool()

	m := martini.Classic()

	m.Map(redisPool)

	// Logger returns a middleware handler that logs the request as it goes in and the response as it goes out.
	m.Use(func() martini.Handler {
		return func(res http.ResponseWriter, req *http.Request, c martini.Context, log *log.Logger) {
			addr := req.Header.Get("X-Real-IP")
			if addr == "" {
				addr = req.Header.Get("X-Forwarded-For")
				if addr == "" {
					addr = req.RemoteAddr
				}
			}

			rw := res.(martini.ResponseWriter)
			c.Next()

			now := time.Now().Format(time.RFC3339)
			glog.Infof("%s [%s] \"%s %s\" %d", addr, now, req.Method, req.URL.Path, rw.Status())
		}
	})

	// Render html templates from directory
	// Support unescaped: https://gist.github.com/techslides/8760361
	m.Use(render.Renderer(render.Options{
		Funcs: []template.FuncMap{
			{
				"formatTime": func(args ...interface{}) string {
					t1 := time.Unix(args[0].(int64), 0)
					return t1.Format(time.Stamp)
				},
				"unescaped": func(args ...interface{}) template.HTML {
					return template.HTML(args[0].(string))
				},
			},
		},
	}))

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

	obmCache := [5]obm.StickerBag{}
	obmCached := [5]bool{false, false, false, false, false}

	done := make(chan bool, 1) // NB: buffered
	go func() {
		for i := 0; i < 5; i++ {
			obmCached[i], obmCache[i] = true, obm.GenStickerBagByLimit(80, i, c_stickers)
		}
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
	}

	ticker := time.NewTicker(time.Minute * 30)
	go func() {
		for _ = range ticker.C {
			for i := 0; i < 5; i++ {
				obmCached[i], obmCache[i] = true, obm.GenStickerBagByLimit(80, i, c_stickers)
			}
		}
	}()

	// Official Sticker
	m.Get("/lines/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		opcode := 0
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.StickerBag{}
		if tag == "" && keyword == "" && obmCached[opcode] == true {
			bag = obm.TrimStickerBagByDetect(detect, obmCache[opcode])
		} else {
			bag = obm.GenStickerBag(detect, c_stickers, opcode, keyword, tag)
		}
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Original Sticker
	m.Get("/clines/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		opcode := 1
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.StickerBag{}
		if tag == "" && keyword == "" && obmCached[opcode] == true {
			bag = obm.TrimStickerBagByDetect(detect, obmCache[opcode])
		} else {
			bag = obm.GenStickerBag(detect, c_stickers, opcode, keyword, tag)
		}
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 25
	m.Get("/dollar25/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		opcode := 2
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.StickerBag{}
		if tag == "" && keyword == "" && obmCached[opcode] == true {
			bag = obm.TrimStickerBagByDetect(detect, obmCache[opcode])
		} else {
			bag = obm.GenStickerBag(detect, c_stickers, opcode, keyword, tag)
		}
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 50
	m.Get("/dollar50/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		opcode := 3
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.StickerBag{}
		if tag == "" && keyword == "" && obmCached[opcode] == true {
			bag = obm.TrimStickerBagByDetect(detect, obmCache[opcode])
		} else {
			bag = obm.GenStickerBag(detect, c_stickers, opcode, keyword, tag)
		}
		if detect.IsMobile() || detect.IsTablet() {
			re.HTML(200, "line", bag)
		} else {
			re.HTML(200, "flat", bag)
		}
	})

	// Sticker with Price 75
	m.Get("/dollar75/", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		opcode := 4
		tag := r.FormValue("tag")
		keyword := r.FormValue("filter")
		detect := mobiledetect.NewMobileDetect(r, nil)
		bag := obm.StickerBag{}
		if tag == "" && keyword == "" && obmCached[opcode] == true {
			bag = obm.TrimStickerBagByDetect(detect, obmCache[opcode])
		} else {
			bag = obm.GenStickerBag(detect, c_stickers, opcode, keyword, tag)
		}
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

	// Detail sticker page
	// http://obmshop.com/detail/%E6%B5%B7%E7%B6%BF%E5%AF%B6%E5%AF%B6%3CBR/%3Eline:/shop/detail/1528
	m.Get("/detail/:s1/:s2/shop/detail/:id", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/detail/" + params["id"]
		http.Redirect(w, r, url, 302)
	})

	// ------------------------------------------------------------------------------------- //

	m.Get("/book/", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		if _, err := os.Stat("/var/opt/www/go/ranklist.csv"); err == nil {
			title, title_link, data := book.GetRankingList()
			bag := TemplateBag{Title: title, Title_link: title_link , Rank: data}
			re.HTML(200, "rank2", bag)
		} else {
			url := "/book/random"
			http.Redirect(w, r, url, 302)
		}
	})

	m.Get("/book/:str", func(params martini.Params, r render.Render, pool book.DbnoPool) {
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
		conn_desc := pool.Get(1)
		defer conn_desc.Close()
		m_ := book.GetBookByList(book.GetBooksByKeywordRedis(keyword, conn_desc), c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		r.HTML(200, "book2", bag)
	})

	m.Get("/book/txt/:str", func(params martini.Params, r render.Render) {
		filename := params["str"]
		name, contents := book.GetTxtContents(filename, c_book)
		bag := TemplateBag{Title: name, List: contents}
		r.HTML(200, "txt", bag)
	})

	m.Get("/book/book_txt/:str", func(params martini.Params, r render.Render) {
		filename := params["str"]
		name, contents := book.GetNovelContents(filename, c_book)
		bag := TemplateBag{Title: name, List: contents}
		r.HTML(200, "txt", bag)
	})

	m.Get("/search-book/", func(w http.ResponseWriter, r *http.Request, re render.Render, pool book.DbnoPool) {
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
		conn_desc := pool.Get(1)
		defer conn_desc.Close()
		m_ := book.GetBookByList(book.GetBooksByKeywordRedis(keyword, conn_desc), c_book)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "book2", bag)
	})

	m.Get("/search-book/:str", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render, pool book.DbnoPool) {
		keyword := params["str"]
		conn_desc := pool.Get(1)
		defer conn_desc.Close()
		m_ :=  book.SearchBook(keyword, stopwords, c_book, c_score, conn_desc)
		bag := TemplateBag{Title: keyword + "を検索", Ary: m_}
		re.HTML(200, "book2", bag)
	})

	m.Post("/search", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Post("/search-book", func(w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/search-book/" + r.FormValue("text")
		http.Redirect(w, r, url, 302)
	})

	m.Get("/search-book/txt/:str", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/book/txt/" + params["str"]
		http.Redirect(w, r, url, 302)
	})
	
	m.Get("/search-book/book_txt/:str", func(params martini.Params, w http.ResponseWriter, r *http.Request, re render.Render) {
		url := "/book/book_txt/" + params["str"]
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

	m.Get("/hack/", func(w http.ResponseWriter, re render.Render) {
		data := hack.GetData()
		bag := TemplateBag{Title: "Hacker News", Data: data}
		re.HTML(200, "hack", bag)
	})

	// ------------------------------------------------------------------------------------- //

	m.Get("/debug/pprof", pprof.Index)
	m.Get("/debug/pprof/cmdline", pprof.Cmdline)
	m.Get("/debug/pprof/profile", pprof.Profile)
	m.Get("/debug/pprof/symbol", pprof.Symbol)
	m.Post("/debug/pprof/symbol", pprof.Symbol)
	m.Get("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	m.Get("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	m.Get("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	m.Get("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)

	m.Run()
}
