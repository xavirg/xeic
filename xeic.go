package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/evanoberholster/imagemeta"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
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
	verbose         bool
	port            string
)

func init() {
	flag.StringVar(&sourcePath, "source", "./", "path for getting files from")
	flag.BoolVar(&removeOriginals, "remove", false, "delete source files")
	flag.StringVar(&destinationPath, "destination", "./output", "path to save the renamed files to")
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.StringVar(&port, "port", "80", "give me a port number")
}

func main() {
	flag.Parse()

	err := filepath.WalkDir(sourcePath, walk)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v file(s) processed, %v file(s) skipped (%v files in total)", processedFiles, skippedFiles, totalFiles)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(destinationPath)))

	log.Printf("starting up server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.IsDir() {
		if verbose {
			log.Printf("reading %v", s)
		}
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
		if verbose {
			log.Printf("processing %s", file)
		}
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
		log.Printf("skipped %v because it has a non-valid extension", file)
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
	if verbose {
		log.Printf("copied %v bytes to %v", bytesWritten, destination)
	}
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

func setupMutualTLS(ca string) *tls.Config {
	clientCACert, err := os.ReadFile(ca)
	if err != nil {
		log.Fatal(err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCertPool,
		MinVersion: tls.VersionTLS12,
	}

	return tlsConfig
}
