package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/kalashnikov/golang_script/ProtobufTest"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net"
	"os"
	"strconv"
)

type Headers []string

const CLIENT_NAME = "GoClient"
const CLIENT_ID = 2
const CLIENT_DESCRIPTION = "Protobuf client"

func main() {
	//filename := flag.String("f", os.Args[1], "Enter the filename to read from")
	dest := flag.String("d", "127.0.0.1:2110", "Enter the destnation socket address")
	flag.Parse()

	// Connect to MongoDB
	session, err := mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Get the collection
	c_stickers := session.DB("obmWeb").C("stickers")
	//c_themes := session.DB("obmWeb").C("themes")

	data, err := GetData(c_stickers)
	//data, err := retrieveDataFromFile(filename)

	checkError(err)
	sendDataToDest(data, dest)
}

func GetData(c *mgo.Collection) ([]byte, error) {
	//STICKERIDINDEX := hdrs.getHeaderIndex("sticker_id")
	//NAMEINDEX := hdrs.getHeaderIndex("name")
	//THUMBNAILINDEX := hdrs.getHeaderIndex("thumbnail")

	ProtoMessage := new(ProtobufTest.TestMessage)
	ProtoMessage.ClientName = proto.String(CLIENT_NAME)
	ProtoMessage.ClientId = proto.Int32(CLIENT_ID)
	ProtoMessage.Description = proto.String(CLIENT_DESCRIPTION)

	var m []bson.M
	//c.Find(bson.M{"id": bson.M{"$lt": 1000000}}).Sort("weigth").Limit(80).All(&m)
	c.Find(bson.M{"id": bson.M{"$gt": 1000000}}).Sort("weigth").Limit(80).All(&m)
	//c.Find(bson.M{"price": 25}).Sort("weigth").Limit(limit).All(&m)
	//c.Find(bson.M{"price": 50}).Sort("weigth").Limit(limit).All(&m)
	//c.Find(bson.M{"price": 75}).Sort("weigth").Limit(limit).All(&m)
	for _, v := range m {
		//Populate items
		testMessageItem := new(ProtobufTest.TestMessage_MsgItem)
		testMessageItem.Id = proto.Int32(int32(v["id"].(int)))
		name, thumbnail := v["name"].(string), v["thumbnail"].(string)
		testMessageItem.Name = &(name)
		testMessageItem.Thumbnail = &(thumbnail)

		ProtoMessage.Messageitems = append(ProtoMessage.Messageitems, testMessageItem)
	}

	//fmt.Println(ProtoMessage.Messageitems)
	return proto.Marshal(ProtoMessage)
}

func retrieveDataFromFile(fname *string) ([]byte, error) {
	file, err := os.Open(*fname)
	checkError(err)
	defer file.Close()

	csvreader := csv.NewReader(file)
	var hdrs Headers
	hdrs, err = csvreader.Read()
	checkError(err)
	STICKERIDINDEX := hdrs.getHeaderIndex("sticker_id")
	NAMEINDEX := hdrs.getHeaderIndex("name")
	THUMBNAILINDEX := hdrs.getHeaderIndex("thumbnail")

	ProtoMessage := new(ProtobufTest.TestMessage)
	ProtoMessage.ClientName = proto.String(CLIENT_NAME)
	ProtoMessage.ClientId = proto.Int32(CLIENT_ID)
	ProtoMessage.Description = proto.String(CLIENT_DESCRIPTION)

	//loop through the records
	for {
		record, err := csvreader.Read()
		if err != io.EOF {
			checkError(err)
		} else {

			break
		}
		//Populate items
		testMessageItem := new(ProtobufTest.TestMessage_MsgItem)
		itemid, err := strconv.Atoi(record[STICKERIDINDEX])
		checkError(err)
		testMessageItem.Id = proto.Int32(int32(itemid))
		testMessageItem.Name = &record[NAMEINDEX]
		testMessageItem.Thumbnail = &record[THUMBNAILINDEX]

		ProtoMessage.Messageitems = append(ProtoMessage.Messageitems, testMessageItem)

		fmt.Println(record)
	}

	//fmt.Println(ProtoMessage.Messageitems)
	return proto.Marshal(ProtoMessage)
}

func sendDataToDest(data []byte, dst *string) {
	conn, err := net.Dial("tcp", *dst)
	checkError(err)
	n, err := conn.Write(data)
	checkError(err)
	fmt.Println("Sent " + strconv.Itoa(n) + " bytes")
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func (h Headers) getHeaderIndex(headername string) int {
	if len(headername) >= 2 {
		for index, s := range h {
			if s == headername {
				return index
			}
		}
	}
	return -1
}
