package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/runtime/v2/logging"
)

func main() {
	logging.Run(log)
}

func log(ctx context.Context, config *logging.Config, ready func() error) error {
	// construct any log metadata for the container
	var wg sync.WaitGroup
	wg.Add(2)
	// forward both stdout and stderr to temp files
	go copy(&wg, config.Stdout, config.ID, "stdout")
	go copy(&wg, config.Stderr, config.ID, "stderr")

	// signal that we are ready and setup for the container to be started
	if err := ready(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func copy(wg *sync.WaitGroup, r io.Reader, id string, kind string) {
	f, _ := os.Create(filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s_%s.log", os.TempDir(), id, kind)))
	defer f.Close()
	defer wg.Done()
	s := bufio.NewScanner(r)
	for s.Scan() {
		f.WriteString(s.Text())
	}
}
