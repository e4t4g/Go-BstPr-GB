package main

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testPipeline struct {
	results map[uint32][]string
}

func (p *testPipeline) SetFileName(file string) error {
	h, err := getHash(file)
	if err != nil {
		hlog.Error("file cannot be open", err)
		return err
	}
	p.results[h] = append(p.results[h], file)

	return nil
}

func (p *testPipeline) FillFileHash(counter map[uint32][]string) {
	for k, v := range p.results {
		counter[k] = v
	}
}

func mastHash(name string) uint32 {
	hash, _ := getHash(name)
	return hash
}

func Test_worker(t *testing.T) {
	type args struct {
		input chan string
		pipe  Pipeline
		wg    *sync.WaitGroup
	}

	tChan := make(chan string, 2)
	tChan <- "main.go"
	tChan <- "main_test.go"
	close(tChan)

	tPipe := testPipeline{results: make(map[uint32][]string)}
	tWg := sync.WaitGroup{}
	tWg.Add(1)

	tMap := map[uint32][]string{
		mastHash("main.go"):      {"main.go"},
		mastHash("main_test.go"): {"main_test.go"},
	}

	tests := []struct {
		name string
		args args
		want map[uint32][]string
	}{
		{
			name: "1 case",
			args: args{
				input: tChan,
				pipe:  &tPipe,
				wg:    &tWg,
			},
			want: tMap,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker(tt.args.input, tt.args.pipe, tt.args.wg)
			tMapEmpty := make(map[uint32][]string)
			tt.args.pipe.FillFileHash(tMapEmpty)
			assert.Equal(t, tMap, tMapEmpty)
		})
	}
}
