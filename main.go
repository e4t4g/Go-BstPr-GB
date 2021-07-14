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

type Pipeline interface {
	FillFileHash(counter map[uint32][]string)
	SetFileName(file string) error
}

type result struct {
	file  string
	crc32 uint32
}

type pipeline struct {
	results chan *result
}

func (p *pipeline) close() {
	close(p.results)
}

func (p *pipeline) SetFileName(file string) error {
	h, err := getHash(file)
	if err != nil {
		hlog.Error("file cannot be open", err)
		return err
	}
	p.results <- &result{
		file:  file,
		crc32: h,
	}
	return nil
}

func (p *pipeline) FillFileHash(counter map[uint32][]string) {
	for result := range p.results {
		counter[result.crc32] = append(counter[result.crc32], result.file)
	}
}

func init() {

	log.SetFormatter(&log.JSONFormatter{})

}

func main() {

	flag.Parse()

	pwd, _ := os.Getwd()
	standardFields := log.Fields{
		"dir": dir,
		"Pwd": pwd,
	}
	hlog = log.WithFields(standardFields)

	fmt.Printf("Searching in %s using %d workers...\n", *dir, *workers)
	input := make(chan string)
	pipe := &pipeline{results: make(chan *result)}
	wg := sync.WaitGroup{}
	wg.Add(*workers)

	for i := 0; i < *workers; i++ {
		go worker(input, pipe, &wg)
	}

	go search(input)

	go func() {
		wg.Wait()
		pipe.close()
	}()

	counter := make(map[uint32][]string)
	pipe.FillFileHash(counter)

	for crc, files := range counter {
		if len(files) > 1 {
			fmt.Printf("\nFound %d duplicates for %v: \n", len(files), crc)
			for count, f := range files {
				hlog.Info("№", count, " File: ", pwd, f)
			}
		}
		if len(files) < 1 {
			fmt.Print("no duplicates")
		}
	}
}

func worker(input chan string, pipe Pipeline, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range input {
		if err := pipe.SetFileName(file); err != nil {
			hlog.Error("file cannot be open", err)
			continue
		}
	}
}

func getHash(filename string) (uint32, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := crc32.NewIEEE()
	_, err = io.Copy(h, f)
	if err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

func search(input chan string) {
	err := filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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
