package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"regexp"
	"strings"
)

func main() {

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c := session.DB("aozora").C("books_go")

	// Use Aggregate to get Author list sorted by book counts
	//db.books_go.aggregate([
	//{ $group: {
	//	"_id": "$author",
	//	"frequency": { "$sum": 1 }
	//}},
	//{'$match':{"frequency": {'$gt': 50}}},
	//{ $sort: { "frequency": 1 } }
	//]).count()
	m := []bson.M{}
	if err = c.Pipe(
		[]bson.M{
			bson.M{
				"$group": bson.M{
					"_id":  "$author",
					"freq": bson.M{"$sum": 1},
				},
			},
			bson.M{
				"$match": bson.M{"freq": bson.M{"$gt": 5}},
			},
			bson.M{
				"$sort": bson.M{"freq": -1},
			},
		},
	).All(&m); err == nil { // Do Query
		ary := make([]string, len(m))
		for _, v := range m {
			ary = append(ary, v["_id"].(string))
		}
		re := regexp.MustCompile(",,+")
		fmt.Println(re.ReplaceAllString(strings.Join(ary, ","), ""))
	}
}
