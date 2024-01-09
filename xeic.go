package main

import (
	"flag"
	"fmt"
	"github.com/evanoberholster/imagemeta"
	"io"
	"io/fs"
	"log"
	"math"
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
	keepOriginal    bool
	processedFiles  int
	totalFiles      int
)

func init() {
	flag.StringVar(&sourcePath, "source", "./", "path for getting files from")
	flag.BoolVar(&keepOriginal, "keep", false, "don't delete source files")
	flag.StringVar(&destinationPath, "destination", "./output", "path to save the renamed files to")
}

func main() {
	flag.Parse()

	err := filepath.WalkDir(sourcePath, walk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("processed %v of %v files in total", processedFiles, totalFiles)
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
		log.Printf("reading %s", file)

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

		if !keepOriginal {
			err := os.Remove(file)
			if err != nil {
				return err
			}
			log.Printf("deleted %s", file)
		}

		processedFiles++
	} else {
		log.Printf("skipping %v (non-valid extension)", file)
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
	sourceFile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		log.Fatal(err)
	}
	defer destinationFile.Close()

	bytesWritten, err := io.Copy(destinationFile, sourceFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s to %s", prettyByteSize(bytesWritten), destination)
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

func prettyByteSize(b int64) string {
	bf := float64(b)
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f%sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1fYiB", bf)
}
