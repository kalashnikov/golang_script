package main

//
// Read-in keywords and scan each document to calculate TF(term frequency)
//
//

import (
	"bufio"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"os"
	"strconv"
	"strings"
	"time"
)

// Readln returns a single line (without the ending \n)
// from the input buffered reader.
// An error is returned iff there is an error with the
// buffered reader.
func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func main() {
	start := time.Now()

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("aozora").C("tf_idf")

	// Open File
	f, ferr := os.Open(os.Args[1])
	if ferr != nil {
		fmt.Printf("Open file failed: %s\n", ferr)
		panic(ferr)
	}

	// Init dfList
	idx := 1
	r := bufio.NewReader(f)
	s, e := Readln(r)
	ary := make([]string, 0)
	for e == nil {
		ary = strings.Split(s, ",")
		s1 := strings.Trim(ary[0], " ")
		s2 := strings.Trim(ary[1], " ")
		if s1 == "" || s2 == "" {
			continue
		}
		if idf, cerr := strconv.ParseFloat(ary[1], 64); cerr == nil {
			// Parallel array
			docs := []int{}       // books id
			tf_idf := []float64{} // score of TF-IDF
			for _, val := range ary[2:] {
				str := strings.Split(val, "_")
				if idx, err := strconv.Atoi(str[0]); err == nil {
					if tf, cerr := strconv.ParseFloat(str[1], 64); cerr == nil {
						docs = append(docs, idx)
						tf_idf = append(tf_idf, tf*idf)
					}
				}
			}
			_, err = c.Upsert(bson.M{"id": idx}, bson.M{"$set": bson.M{
				"id":        idx,
				"word":      ary[0],
				"docs":      docs,
				"score":     tf_idf,
				"update_at": time.Now(),
			},
			},
			)

			if err != nil {
				panic(err)
			}
		} else if idf != 0 {
			fmt.Println(idx)
			panic(cerr)
		}
		idx++
		s, e = Readln(r)
	}

	f.Close()

	// Check the result
	//var m []bson.M
	//_ = c.Find(nil).All(&m)
	//for _, v := range m {
	//	fmt.Println(v)
	//}

	fmt.Printf("Time used: %v", time.Since(start))
}
