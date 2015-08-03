package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	t := int64(0)

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		//fmt.Printf("Extract %s ...\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			return (err)
		}

		// Create new file
		path := filepath.Join("", f.Name)
		of, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		defer of.Close()

		n, err := io.Copy(of, rc)
		if err != nil && n != 0 {
			fmt.Printf("# Copy ERROR: %s %d | Total: %d \n", f.Name, n, t)
			return (err)
		}

		t += n
		rc.Close()
	}

	return nil
}

func main() {
	err := Unzip(os.Args[1], "")
	fmt.Println(err)
}
