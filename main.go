package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type finder struct {
	finds 		chan searchItem
	semaphore 	chan struct{}
	wg			*sync.WaitGroup
}

type searchItem struct {
	name string
	path string
}

func main() {
	var (
		root = flag.Bool("p", false, "use current position as root")
		isLinux = flag.Bool("l", false, "Is your system Linux")
	)
	flag.Parse()

	var exPath string
	if *root {
		path, err := os.Executable()
		if err != nil {
			panic(err)
		}

		exPath = filepath.Dir(path)
	} else {
		if *isLinux {
			exPath = "/"
		} else {
			exPath = "C:"
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		query, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		query = strings.TrimSpace(query)
		if query == "q" {
			return
		}

		f := &finder{
			finds:     make(chan searchItem),
			semaphore: make(chan struct{}, 100),
			wg:        new(sync.WaitGroup),
		}

		go func() {
			for result := range f.finds {
				fmt.Printf("file: %s, exist in: %s\n", result.name, result.path)
			}
		}()

		f.semaphore <- struct{}{}
		f.wg.Add(1)
		go f.walkTo(exPath, query)

		f.wg.Wait()
		close(f.finds)
	}
}

func (f *finder) walkTo(path, query string) {
	defer f.wg.Done()

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			select{
			case f.semaphore <- struct{}{}:
				f.wg.Add(1)
				go func() {
					defer func() {<- f.semaphore}()
					f.walkTo(filepath.Join(path, entry.Name()), query)
				}()

			default:
				f.wg.Add(1)
				f.walkTo(filepath.Join(path, entry.Name()), query)
			}
		} else {
			if entry.Name() == query {
				f.finds <- searchItem{
					name: entry.Name(),
					path: path,
				}
			}
		}
	}
}