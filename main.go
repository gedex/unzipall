package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// walkFiles starts a goroutine to walk the directory tree at root and send the
// path of each zip file on the string channel. It sends the result of the walk
// on the error channel. If done is closed, walkFiles abandons its work.
func walkFiles(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	paths := make(chan string)
	errc := make(chan error, 1)
	go func() {
		// Close the paths channel after Walk returns.
		defer close(paths)

		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(info.Name(), ".zip") {
				return nil
			}
			select {
			case paths <- path:
			case <-done:
				return errors.New("walk canceled")
			}
			return nil
		})
	}()
	return paths, errc
}

// result is the extracted zip.
type result struct {
	zip       string
	extracted string
	err       error
}

// unzipper reads path names from paths and sends extracted path name of corresponding
// files, rooted at dstRoot, on resc until either paths or done is closed.
func unzipper(dstRoot string, done <-chan struct{}, paths <-chan string, resc chan<- result) {
	for path := range paths {
		target := unzipTarget(path, dstRoot)
		if err := os.MkdirAll(target, 0755); err != nil {
			resc <- result{path, target, err}
		}
		fmt.Printf("Unzip %s to %s\n", path, target)
		err := exec.Command("unzip", "-o", path, "-d", target).Run()
		select {
		case resc <- result{path, target, err}:
		case <-done:
			return
		}
	}
}

func unzipTarget(path, dstRoot string) string {
	rel, _ := filepath.Rel(*src, path)
	base := filepath.Base(rel)
	ext := filepath.Ext(rel)

	return filepath.Join(dstRoot, filepath.Dir(rel), base[:len(base)-len(ext)])
}

// UnzipAll reads all the files in the file tree rooted at srcRoot and returns a
// map from zip path to the extracted path. If the directory walk fails or any
// read operation fails, UnzipAll returns an error. In that case, UnzipAll does
// not wait for inflight read operations to complete.
func UnzipAll(srcRoot, dstRoot string) (map[string]string, error) {
	// UnzipAll closes the done channel when it returns; it may do so before
	// receiving all the values from c and errc.
	done := make(chan struct{})
	defer close(done)

	paths, errc := walkFiles(done, srcRoot)

	// Start a fixed number of goroutines to read and unzip files.
	resc := make(chan result)
	var wg sync.WaitGroup
	const numUnzippers = 20
	wg.Add(numUnzippers)
	for i := 0; i < numUnzippers; i++ {
		go func() {
			unzipper(dstRoot, done, paths, resc)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(resc)
	}()

	m := make(map[string]string)
	for r := range resc {
		if r.err != nil {
			return nil, r.err
		}
		m[r.zip] = r.extracted
	}
	// Check whether the Walk failed.
	if err := <-errc; err != nil {
		return nil, err
	}
	if len(m) == 0 {
		fmt.Printf("No zip file found in %s\n", srcRoot)
	}
	return m, nil
}

var (
	src = flag.String("src", ".", "source directory containing zip files")
	dst = flag.String("dst", ".", "destination directory for extracted zip files")
)

func main() {
	flag.Parse()

	_, err := UnzipAll(*src, *dst)
	if err != nil {
		fmt.Println(err)
		return
	}
}
