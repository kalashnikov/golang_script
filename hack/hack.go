package hack

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

type News struct {
	Title string
	Url   string
	CUrl  string
	Score int
}

func GetData() []News {
	file, err := os.Open("/var/opt/www/go/hackernews.csv")
	CheckError(err)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var data []News
	csvreader := csv.NewReader(file)
	for {
		record, err := csvreader.Read()
		if err != io.EOF {
			CheckError(err)
		} else {
			break
		}
		score, err := strconv.Atoi(record[0])
		CheckError(err)
		news := News{Title: record[1], Score: score, Url: record[2], CUrl: record[3]}
		data = append(data, news)
	}
	return data
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
