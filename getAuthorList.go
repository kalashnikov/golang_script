package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
				"$match": bson.M{"freq": bson.M{"$gt": 10}},
			},
			bson.M{
				"$sort": bson.M{"freq": -1},
			},
		},
	).All(&m); err == nil { // Do Query
		for _, v := range m {
			fmt.Printf("%s,%d\n", v["_id"].(string), v["freq"].(int))
		}
	}
}
