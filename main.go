package main

import (
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	dir     = flag.String("dir", ".", "directory")
	workers = flag.Int("workers", runtime.NumCPU(), "num of workers")
	hlog    *log.Entry
)

type result struct {
	file  string
	crc32 [32]byte
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {

	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		hlog.Error("problem with file", err)
	}

	standardFields := log.Fields{
		"dir": dir,
		"Pwd": pwd,
	}
	hlog = log.WithFields(standardFields)

	fmt.Printf("Searching in %s using %d workers...\n", *dir, *workers)
	input := make(chan string)
	results := make(chan *result)

	wg := sync.WaitGroup{}
	wg.Add(*workers)

	for i := 0; i < *workers; i++ {
		go worker(input, results, &wg)
	}

	go search(input)

	go func() {
		wg.Wait()
		close(results)
	}()

	counter := make(map[[32]byte][]string)
	for result := range results {
		counter[result.crc32] = append(counter[result.crc32], result.file)
	}

	for crc, files := range counter {
		if len(files) > 1 {
			fmt.Printf("Found %d duplicates for %v: \n", len(files), crc32.ChecksumIEEE(crc[:]))
			for count, f := range files {
				//fmt.Printf("%v %s \n", count, f)
				hlog.Info("№", count, " File: ", pwd, f)
			}
		}
		if len(files) < 1 {
			fmt.Print("no duplicates")
		}
	}

}

func worker(input chan string, results chan<- *result, wg *sync.WaitGroup) {

	for file := range input {
		h := crc32.NewIEEE()
		var sum [32]byte
		f, err := os.Open(file)
		if err != nil {
			hlog.Error("file cannot be open", err)
			//fmt.Fprintln(os.Stderr, err)
			continue
		}
		if _, err = io.Copy(h, f); err != nil {
			hlog.Error("problem with file", err)
			//fmt.Fprintln(os.Stderr, err)
			f.Close()
			continue
		}
		f.Close()
		copy(sum[:], h.Sum(nil))
		results <- &result{
			file:  file,
			crc32: sum,
		}
	}
	wg.Done()
}

func search(input chan string) {
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//fmt.Fprintln(os.Stderr, err)
			hlog.Error("incorrect directory path", err)
		} else if info.Mode().IsRegular() {
			input <- path
		}
		return nil
	})
	if err != nil {
		hlog.Error("incorrect directory path", err)
	}

	close(input)
}
