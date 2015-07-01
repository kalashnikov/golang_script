package obm

import (
	//"fmt"
	"github.com/Shaked/gomobiledetect"
	"github.com/kalashnikov/golang_script/utility"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"math/rand"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ResultArray []bson.M

type StickerBag struct {
	Title string
	Ary   ResultArray
	List  []string
}

type StickerDetailBag struct {
	Id          int
	Title       string
	Name        string
	Detail      string
	Thumbnail   string
	Description string
	DetailImg   []string
	Price       int
	Tags        []string
}

func GetLimitByPlatform(detect *mobiledetect.MobileDetect) int {
	limit := 80
	if detect.IsMobile() {
		limit = 20
	} else if detect.IsTablet() {
		limit = 40
	}
	return limit
}

// Use tag file to speed up
// Assumption: Tag editing feature is disabled.
func GetTags(c *mgo.Collection) []string {
	m_ := []string{}
	m_ = append(m_, "隨機")
	if _, err := os.Stat("/var/opt/www/go/tags.txt"); err == nil {
		if b, err := ioutil.ReadFile("/var/opt/www/go/tags.txt"); err == nil {
			for _, v := range strings.Split(string(b), ",") {
				m_ = append(m_, v)
			}
		}
	} else {
		m := []string{}
		if err := c.Find(nil).Distinct("tag", &m); err == nil { // Do Query
			for _, v := range m {
				if v == "" || strings.Contains(v, "http") ||
					strings.Contains(v, "line") ||
					strings.Contains(v, " ") ||
					len(v) < 2 {
					continue
				}
				m_ = append(m_, v)
			}
		}
	}
	return m_
}

// Do the data retrieval and return result
func GetStickers(limit int, c *mgo.Collection, op_type int) []bson.M {
	m := []bson.M{}
	if op_type == 0 {
		c.Find(bson.M{"id": bson.M{"$lt": 1000000}}).Sort("weigth").Limit(limit).All(&m)
	} else if op_type == 1 {
		c.Find(bson.M{"id": bson.M{"$gt": 1000000}}).Sort("weigth").Limit(limit).All(&m)
	} else if op_type == 2 {
		c.Find(bson.M{"price": 25}).Sort("weigth").Limit(limit).All(&m)
	} else if op_type == 3 {
		c.Find(bson.M{"price": 50}).Sort("weigth").Limit(limit).All(&m)
	} else if op_type == 4 {
		c.Find(bson.M{"price": 75}).Sort("weigth").Limit(limit).All(&m)
	} else {
		c.Find(nil).Sort("weigth").Limit(limit).All(&m)
	}
	return m
}

// For keyword searching
func GetStickersByKeyword(keyword string, c *mgo.Collection) []bson.M {
	m := []bson.M{}
	c.Find(bson.M{
		"$or": []interface{}{
			bson.M{"name": &bson.RegEx{Pattern: keyword, Options: "i"}},
			bson.M{"description": &bson.RegEx{Pattern: keyword, Options: "i"}},
			bson.M{"alias": &bson.RegEx{Pattern: keyword, Options: "i"}},
		},
	}).Sort("weigth").Limit(50).All(&m)
	return m
}

// For tag searching
func GetStickersByTag(tag string, c *mgo.Collection) []bson.M {
	m := []bson.M{}
	if tag == "隨機" {
		c.Find(nil).Sort("random").Limit(25).All(&m)
		go UpdateStickerRandomField(c)
	} else {
		c.Find(bson.M{"tag": tag}).Sort("weigth").Limit(50).All(&m)
	}
	return m
}

func UpdateStickerRandomField(c *mgo.Collection) {
	m := []bson.M{}
	c.Find(nil).Sort("random").Limit(25).All(&m) // Get Top100 Sorted by Random
	for _, v := range m {
		id := v["id"]
		v["random"] = rand.Int() % 100000000
		v["update_at"] = time.Now()
		_, err := c.Upsert(bson.M{"id": id}, bson.M{"$set": v})
		if err != nil {
			panic(err)
		}
	}
}

// Get the sticker data by ID and init bag for /detail/ pag
func GetStickersDetail(id string, c_stickers, c_themes *mgo.Collection) StickerDetailBag {
	m := bson.M{}
	bag := StickerDetailBag{}

	// For Sticker, id is Int.
	// For Theme, id is String.
	if idInt, err := strconv.Atoi(id); err == nil {
		c_stickers.Find(bson.M{"id": idInt}).One(&m)

		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.Int {
			GetStickerInfo(id, c_stickers, c_themes)
		}

		bag.Id = m["id"].(int)
	} else {
		c_themes.Find(bson.M{"id": id}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.String {
			GetStickerInfo(id, c_stickers, c_themes)
		}
		bag.Id, _ = strconv.Atoi(m["id"].(string))
	}

	bag.Name = m["name"].(string)
	bag.Detail = m["detail"].(string)
	bag.Thumbnail = m["thumbnail"].(string)
	bag.Description = m["description"].(string)

	// Detail image may be list
	imgList := make([]string, 0)
	v := reflect.ValueOf(m["detailImg"])
	if v.Kind() == reflect.String {
		imgList = append(imgList, m["detailImg"].(string))
	} else {
		imgs := v
		if imgs.Kind() != 0 {
			for i := 0; i < imgs.Len(); i++ {
				str := imgs.Index(i).Interface().(string)
				imgList = append(imgList, str)
			}
		}
	}
	bag.DetailImg = imgList

	// Try to convert into Int
	v = reflect.ValueOf(m["price"])
	if v.Kind() == reflect.Int {
		bag.Price = v.Interface().(int)
	} else if v.Kind() == reflect.Float64 {
		bag.Price = int(v.Interface().(float64))
	}

	// Tag might be empty, need checking
	tagList := make([]string, 0)
	tags := reflect.ValueOf(m["tag"])
	if tags.Kind() != 0 {
		for i := 0; i < tags.Len(); i++ {
			str := tags.Index(i).Interface().(string)
			tagList = append(tagList, str)
		}
	}
	bag.Tags = tagList

	bag.Title = bag.Name + " | 歐貝賣專業代購"
	return bag
}

func GenStickerBag(detect *mobiledetect.MobileDetect, c_stickers *mgo.Collection,
	op_type int, keyword string, tag string) StickerBag {
	limit := GetLimitByPlatform(detect)
	tags := GetTags(c_stickers)
	m := []bson.M{}
	if keyword != "" {
		m = GetStickersByKeyword(keyword, c_stickers)
	} else if tag != "" {
		m = GetStickersByTag(tag, c_stickers)
	} else {
		m = GetStickers(limit, c_stickers, op_type)
	}
	return StickerBag{Title: "LINE 貼圖| 歐貝賣專業代購", Ary: m, List: tags}
}

// Download Sticker Information from Official Site
// Insert data into database if successful and return true
// Return false if something wrong
// TODO: Need further improve header/cookies for some stickers
func GetStickerInfo(idStr string, c_stickers, c_themes *mgo.Collection) bool {
	rand.Seed(time.Now().UTC().UnixNano())

	c := c_stickers

	// Determine this is a sticker or theme
	urlStr, id, theme := "", 0, 0
	if v, err := strconv.Atoi(idStr); err == nil {
		id, theme = v, 0
		urlStr = "https://store.line.me/stickershop/product/" + idStr + "/zh-Hant"
	} else {
		theme = 1
		c = c_themes
		urlStr = "https://store.line.me/themeshop/product/" + idStr + "/zh-Hant"
	}

	gCookieJar, _ := cookiejar.New(nil)

	priceTable := make(map[float64]float64, 3)
	priceTable[30] = 25
	priceTable[60] = 50
	priceTable[90] = 75

	// Get cookie
	urlBase := "https://store.line.me/stickershop/"
	cookies := geturl.TestGet(urlBase, gCookieJar)

	// Set the gCookieJar
	u, _ := url.Parse(urlStr)
	gCookieJar.SetCookies(u, cookies)

	// Use Cookie (Broken - no difference)
	contents := geturl.GetDocByURLWithCookie(urlStr, gCookieJar)

	// Check data exist
	desc := geturl.GetSingleText(contents, "p.mdMN07Desc")
	if desc == "" {
		return false
	}

	// Set Price
	dprice, err := strconv.ParseFloat(geturl.GetSingleText(contents, "p.mdMN05Price")[3:], 64)
	if err != nil {
		panic(err)
	} else {
		dprice = priceTable[dprice*0.25]
	}

	imgtext := geturl.GetSingleText(contents, "h2.mdMN05Ttl")
	imgsrc := geturl.GetFirstAttr(contents, "div.mdMN05Img img", "src")

	// Detail Image may be more than one
	detailImg := make([]string, 1)
	if theme == 0 {
		detailImg = append(detailImg, geturl.GetFirstAttr(contents, "div.mdMN07Img img", "src"))
	} else {
		detailImg = geturl.GetAttrs(contents, "li.mdMN07Li img", "src")
	}

	random := rand.Int() % 100000000

	if theme == 0 {
		_, err = c.Upsert(bson.M{"id": id}, bson.M{"$set": bson.M{
			"id":          id,
			"sticker_id":  id,
			"name":        imgtext,
			"tag":         make([]string, 0),
			"detail":      urlStr,
			"description": desc,
			"price":       dprice,
			"thumbnail":   imgsrc,
			"weigth":      0,
			"random":      random,
			"detailImg":   detailImg,
			"update_at":   time.Now(),
			"create_at":   time.Now(),
		},
		},
		)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = c.Upsert(bson.M{"id": id}, bson.M{"$set": bson.M{
			"id":          idStr,
			"sticker_id":  idStr,
			"name":        imgtext,
			"tag":         make([]string, 0),
			"detail":      urlStr,
			"description": desc,
			"price":       dprice,
			"thumbnail":   imgsrc,
			"weigth":      0,
			"random":      random,
			"detailImg":   detailImg,
			"update_at":   time.Now(),
			"create_at":   time.Now(),
		},
		},
		)
		if err != nil {
			panic(err)
		}
	}

	return true
}
