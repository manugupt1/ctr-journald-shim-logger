package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/coreos/go-systemd/journal"
	"golang.org/x/sys/unix"
)

var f, _ = os.OpenFile("/tmp/logger", os.O_RDWR|os.O_CREATE, 0755)

// Config of the container logs
type Config struct {
	ID        string
	Namespace string
	Stdout    io.Reader
	Stderr    io.Reader
}

// LoggerFunc is implemented by custom v2 logging binaries
type LoggerFunc func(context.Context, *Config, func() error) error

func main() {
	f, _ := os.OpenFile("/tmp/loggerq", os.O_RDWR|os.O_CREATE, 0755)
	defer f.Close()
	f.WriteString(fmt.Sprintf("%s\n", os.Environ()))
	f.WriteString("env written")
	wd, err := os.Getwd()
	if err != nil {
		f.WriteString("errro is " + err.Error())
	}

	f.WriteString(wd + "\n")
	// logging.Run(log)
	Run(log)
	f.WriteString("copied")
}

func log(ctx context.Context, config *Config, ready func() error) error {
	// construct any log metadata for the container
	vars := map[string]string{
		"SYSLOG_IDENTIFIER": fmt.Sprintf("%s:%s", config.Namespace, config.ID),
	}
	var wg sync.WaitGroup
	wg.Add(2)
	// forward both stdout and stderr to the journal
	go copy(&wg, config.Stdout, journal.PriInfo, vars)
	go copy(&wg, config.Stderr, journal.PriErr, vars)

	// signal that we are ready and setup for the container to be started
	if err := ready(); err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func copy(wg *sync.WaitGroup, r io.Reader, pri journal.Priority, vars map[string]string) {
	defer wg.Done()
	s := bufio.NewScanner(r)
	journal.Send(fmt.Sprintf("%s", os.Environ()), journal.PriInfo, vars)
	for s.Scan() {
		journal.Send(s.Text(), pri, vars)
	}
}

// Run the logging driver
func Run(fn LoggerFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := &Config{
		ID:        os.Getenv("CONTAINER_ID"),
		Namespace: os.Getenv("CONTAINER_NAMESPACE"),
		Stdout:    os.NewFile(3, "CONTAINER_STDOUT"),
		Stderr:    os.NewFile(4, "CONTAINER_STDERR"),
	}
	var (
		sigCh = make(chan os.Signal, 32)
		errCh = make(chan error, 1)
		wait  = os.NewFile(5, "CONTAINER_WAIT")
	)
	finfo, err := os.Readlink("/proc/self/fd/3")
	f.WriteString("fInfi is: " + finfo + "err:" + err.Error() + "\n")
	w, _ := filepath.Abs(f.Name())
	cwd, _ := os.Getwd()
	f.WriteString("config written:" + wait.Name() + "--------------------" + w)
	f.WriteString("\nwd is: " + cwd + "\n")
	dirs, _ := os.ReadDir(cwd)
	for _, dir := range dirs {
		f.WriteString(fmt.Sprintf("\n%s\n", dir.Name()))
	}

	signal.Notify(sigCh, unix.SIGTERM)

	go func() {
		errCh <- fn(ctx, config, wait.Close)
	}()

	for {
		select {
		case <-sigCh:
			cancel()
		case err := <-errCh:
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

}
