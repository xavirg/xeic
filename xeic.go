package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/evanoberholster/imagemeta"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CustomTimeFormat = "2006-01-02_15.04.05"
)

var (
	sourcePath      string
	destinationPath string
	removeOriginals bool
	processedFiles  int
	skippedFiles    int
	totalFiles      int
)

func init() {
	flag.StringVar(&sourcePath, "source", "./", "path for getting files from")
	flag.BoolVar(&removeOriginals, "remove", false, "delete source files")
	flag.StringVar(&destinationPath, "destination", "./output", "path to save the renamed files to")
}

func main() {
	flag.Parse()

	log.Printf("looking for picture files at %s", sourcePath)

	err := filepath.WalkDir(sourcePath, walk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v file(s) processed, %v file(s) skipped (%v files in total)", processedFiles, skippedFiles, totalFiles)
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {

		err = processFile(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func processFile(file string) error {
	totalFiles++
	extension := strings.ToLower(filepath.Ext(file))
	if isValidFile(extension) {
		timestamp, err := getPictureDateTime(file)
		if err != nil {
			return err
		}

		if timestamp.IsZero() {
			log.Printf("%s metadata does not have a valid timestamp", file)
			return nil
		}

		newPath := filepath.Join(destinationPath, fmt.Sprintf("%s%s", timestamp.Format(CustomTimeFormat), extension))

		err = os.MkdirAll(destinationPath, os.ModePerm)
		if err != nil {
			return err
		}

		err = copyFile(file, newPath)
		if err != nil {
			return err
		}

		if removeOriginals {
			err := os.Remove(file)
			if err != nil {
				return err
			}
			log.Printf("deleted %s", file)
		}
		processedFiles++
	} else {
		skippedFiles++
	}
	return nil
}

func getPictureDateTime(file string) (time.Time, error) {
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	e, err := imagemeta.Decode(f)
	if err != nil {
		panic(err)
	}
	return e.DateTimeOriginal(), err
}

func copyFile(src string, destination string) error {
	if fileExists(destination) {
		errMsg := fmt.Sprintf("file %s already exists", destination)
		return errors.New(errMsg)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	log.Printf("processed %s to %s", src, destination)

	return nil
}

func isValidFile(extension string) bool {
	switch extension {
	case
		".heic",
		".jpeg",
		".jpg":
		return true
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Open(path) // For read access.
	return err == nil

}
