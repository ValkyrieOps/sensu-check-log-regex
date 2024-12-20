package main

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"sync"
	"fmt"
)

const bufSize = 1000

type Analyzer struct {
	Procs     int
	Log       io.Reader
	Func      AnalyzerFunc
	bytesRead int64
	wg        sync.WaitGroup
}

type discardInterface interface {
	io.Writer
	io.ReaderFrom
}

type DiscardWriter struct {
	discardWriter discardInterface
	BytesRead     int64
}

func (d *DiscardWriter) Write(b []byte) (int, error) {
	n, err := d.discardWriter.Write(b)
	d.BytesRead += int64(n)
	return n, err
}

func (d *DiscardWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := d.discardWriter.ReadFrom(r)
	d.BytesRead += n
	return n, err
}

func NewDiscardWriter() *DiscardWriter {
	discard := ioutil.Discard.(discardInterface)
	return &DiscardWriter{discardWriter: discard}
}

type AnalyzerFunc func([]byte) *Result

type Result struct {
	Match string `json:"match"`
	Err   error  `json:"error,omitempty"`
}

func (a *Analyzer) Go(ctx context.Context) <-chan Result {
	resultC := make(chan Result)
	a.wg.Add(a.Procs)
	producer := a.startProducer(ctx)
	go func() {
		a.wg.Wait()
		close(resultC)
	}()
	for i := 0; i < a.Procs; i++ {
		go a.consumer(ctx, producer, resultC)
	}
	return resultC
}

func (a *Analyzer) BytesRead() int64 {
	a.wg.Wait()
	return a.bytesRead
}

func (a *Analyzer) startProducer(ctx context.Context) <-chan []byte {
	result := make(chan []byte, bufSize)
	reader := bufio.NewReaderSize(a.Log, 32*1024*1024)
	discard := NewDiscardWriter()
	teeReader := io.TeeReader(reader, discard)
	scanner := bufio.NewScanner(teeReader)
	var wg sync.WaitGroup
	wg.Add(1)
	a.wg.Add(1)
	go func() {
		defer wg.Done()
		for scanner.Scan() {
			line := scanner.Bytes()
			// Copy the line, since the scanner can reclaim it
			lineCopy := make([]byte, len(line))
			copy(lineCopy, line)
			select {
			case <-ctx.Done():
				close(result)
				return
			case result <- lineCopy:
			}
		}
		if err := scanner.Err(); err != nil {
//			fatal("error while scanning log: %s", err)
			fmt.Println("CRITICAL\nError encoding result buffer")
//                                return sensu.CheckStateCritical, nil
		}
		close(result)
	}()
	go func() {
		defer a.wg.Done()
		wg.Wait()
		a.bytesRead = discard.BytesRead
	}()
	return result
}

func (a *Analyzer) consumer(ctx context.Context, producer <-chan []byte, results chan<- Result) {
	defer a.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-producer:
			if !ok {
				return
			}
			result := a.Func(line)
			if result != nil {
				select {
				case results <- *result:
				case <-ctx.Done():
				}
			}
		}
	}
}

func NoopAnalyzerFunc(line []byte) *Result {
	return nil
}
