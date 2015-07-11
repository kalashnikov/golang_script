package obm

import (
	"fmt"
	"github.com/Shaked/gomobiledetect"
	"github.com/garyburd/redigo/redis"
	_ "github.com/golang/groupcache"
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
	"sync"
	"time"
)

type ResultArray []bson.M

// a pool embedding the original pool and adding adbno state
type DbnoPool struct {
	redis.Pool
}

// "overriding" the Get method
func (p *DbnoPool) Get(dbId int) redis.Conn {
	conn := p.Pool.Get()
	conn.Do("SELECT", dbId)
	return conn
}

func InitRedisPool() DbnoPool {
	pool2 := DbnoPool{
		redis.Pool{
			MaxIdle:   80,
			MaxActive: 12000, // max number of connections
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", ":6379")
				if err != nil {
					panic(err.Error())
				}
				return c, err
			},
		},
	}
	return pool2
}

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

func GetStickerByWeigth(limit int, c *mgo.Collection, conn redis.Conn, op_type int) []bson.M {
	keys := make([]int, 0)
	if op_type == 0 {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set0", 0, limit))
	} else if op_type == 1 {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set1", 0, limit))
	} else if op_type == 2 {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set2", 0, limit))
	} else if op_type == 3 {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set3", 0, limit))
	} else if op_type == 4 {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set4", 0, limit))
	} else {
		keys, _ = redis.Ints(conn.Do("ZRANGE", "set0", 0, limit))
	}

	m := []bson.M{}
	m_ := []bson.M{}
	if err := c.Find(bson.M{"id": bson.M{"$in": keys}}).All(&m); err == nil { // Do Query
		for _, v := range m {
			m_ = append(m_, v)
		}
	}

	// Sort into same order of docList
	// O(n^2) but only size=20....
	m__ := []bson.M{}
	for _, v := range keys {
		for _, d := range m_ {
			if v == d["id"].(int) {
				m__ = append(m__, d)
			}
		}
	}
	return m__
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
		go UpdateStickerRandomField(m, c)
	} else {
		c.Find(bson.M{"tag": tag}).Sort("weigth").Limit(50).All(&m)
	}
	return m
}

func UpdateStickerRandomField(m []bson.M, c *mgo.Collection) {
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

// Decrease the weight in each view
// This operation cost ~5ms for update
func DecreaseStickerWeigth(m bson.M, c *mgo.Collection) {
	id := m["id"]

	val := m["weigth"]
	v := reflect.ValueOf(m["weigth"])
	if v.Kind() == reflect.Int {
		val = float64(m["weigth"].(int)) - 0.2
	} else if v.Kind() == reflect.Float64 {
		val = m["weigth"].(float64) - 0.2
	} else {
		val = 0
	}
	m["weigth"] = val
	m["update_at"] = time.Now()
	_, err := c.Upsert(bson.M{"id": id}, bson.M{"$set": m})
	if err != nil {
		panic(err)
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
		go DecreaseStickerWeigth(m, c_stickers)
	} else {
		c_themes.Find(bson.M{"id": id}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.String {
			GetStickerInfo(id, c_stickers, c_themes)
		}
		bag.Id, _ = strconv.Atoi(m["id"].(string))
		go DecreaseStickerWeigth(m, c_themes)
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

func TrimStickerBagByDetect(detect *mobiledetect.MobileDetect, bag StickerBag) StickerBag {
	limit := GetLimitByPlatform(detect)
	if limit-1 < len(bag.Ary) {
		bag.Ary = bag.Ary[0 : limit-1]
	}
	return bag
}

// Generate sticker bag for template,
// Limit used. Mainly for Cache Array
func GenStickerBagByLimit(limit, op_type int, c_stickers *mgo.Collection) StickerBag {
	tags := GetTags(c_stickers)
	m := GetStickers(limit, c_stickers, op_type)
	return StickerBag{Title: "LINE 貼圖| 歐貝賣專業代購", Ary: m, List: tags}
}

// Generate sticker bag for template, dynamic generation by input
func GenStickerBag(detect *mobiledetect.MobileDetect, c_stickers *mgo.Collection, //conn redis.Conn,
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
		//m = GetStickerByWeigth(limit, c_stickers, conn, op_type)
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

func GetStickersData(id string, c_stickers, c_themes *mgo.Collection, wg *sync.WaitGroup) {
	defer wg.Done()

	m := bson.M{}

	// For Sticker, id is Int.
	// For Theme, id is String.
	if idInt, err := strconv.Atoi(id); err == nil {
		c_stickers.Find(bson.M{"id": idInt}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.Int {
			ok := GetStickerInfo(id, c_stickers, c_themes)
			if ok {
				fmt.Printf("### Get %s ... %t\n", id, ok)
			}
		}
	} else {
		c_themes.Find(bson.M{"id": id}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.String {
			ok := GetStickerInfo(id, c_stickers, c_themes)
			if ok {
				fmt.Printf("### Get %s ... %t\n", id, ok)
			}
		}
	}
}
