package main

import (
	"fmt"
	"github.com/kalashnikov/golang_script/obm"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"runtime"
	"strconv"
	"sync"
)

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

	m := []int{}
	list := make([]int, 0)

	// Creator stickers
	c_stickers.Find(bson.M{"id": bson.M{"$gt": 1000000}}).Distinct("id", &m)
	start1 := m[len(m)-100]
	for i := start1 + 1; i < start1+500; i++ {
		list = append(list, i)
	}

	// Get the data
	cpunum := runtime.NumCPU()
	var wg sync.WaitGroup
	for i := 0; i < cpunum; i++ {
		wg.Add(1)
		go func() {
			for _, i := range list {
				GetStickersData(strconv.Itoa(i), c_stickers, c_themes)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func GetStickersData(id string, c_stickers, c_themes *mgo.Collection) {
	m := bson.M{}

	// For Sticker, id is Int.
	// For Theme, id is String.
	if idInt, err := strconv.Atoi(id); err == nil {
		c_stickers.Find(bson.M{"id": idInt}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.Int {
			ok := obm.GetStickerInfo(id, c_stickers, c_themes)
			if ok {
				fmt.Printf("### Get %s ... %t\n", id, ok)
			}
		}
	} else {
		c_themes.Find(bson.M{"id": id}).One(&m)
		// If not found, try to get it
		v := reflect.ValueOf(m["id"])
		if v.Kind() != reflect.String {
			ok := obm.GetStickerInfo(id, c_stickers, c_themes)
			if ok {
				fmt.Printf("### Get %s ... %t\n", id, ok)
			}
		}
	}
}
