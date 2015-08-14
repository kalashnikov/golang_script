package main

import (
	"fmt"
	// "golang.org/x/image/webp"
	"github.com/chai2010/webp"
	_ "image"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"sync"
)

func main() {

	// Ref: How would you define a pool of goroutines to be executed at once in Golang?
	//  http://stackoverflow.com/questions/18405023/how-would-you-define-a-pool-of-goroutines-to-be-executed-at-once-in-golang
	tasks := make(chan string, 64)

	// spawn four worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			for file := range tasks {
				DoConvert(file)
			}
			wg.Done()
		}()
	}

	if a, err := filepath.Glob("gpics/*"); err == nil {
		for _, file := range a {
			tasks <- file
		}
	}
	close(tasks)

	wg.Wait()
}

func DoConvert(file string) {
	// Read image from file that already exists
	existingImageFile, err := os.Open(file)
	if err != nil {
		// Handle error
	}
	defer existingImageFile.Close()

	//img, _, err := image.Decode(existingImageFile)
	//img, err := webp.Decode(existingImageFile)
	//if err != nil {
	//	fmt.Printf("Error during decode: %s, %s \n", err, file)
	//	return
	//}

	// Decode webp
	img, err := webp.Decode(existingImageFile)
	if err != nil {
		fmt.Printf("Error during decode: %s, %s \n", err, file)
		return
	}

	name := "gpics_png/" + path.Base(file) + ".png"
	out, err := os.Create(name)
	if err != nil {
		panic(err)
	}

	err = png.Encode(out, img)
	if err != nil {
		panic(err)
	} else {
		fmt.Printf("### Finished: %s\n", file)
	}
}
