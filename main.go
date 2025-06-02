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
	root	string
	finds 	chan searchItem
	wp 		*workerPool
}

type searchItem struct {
	name string
	path string
}

type workerPool struct {
	tasks 		chan func()
	tasksQueue 	chan func()
	wg 			*sync.WaitGroup
	quit		chan struct{}
}

func newWorkerPool(poolSize int) *workerPool {
	wp := &workerPool{
		tasks: make(chan func(), poolSize),
		tasksQueue: make(chan func(), poolSize * 10),
		wg: new(sync.WaitGroup),
		quit: make(chan struct{}),
	}
	go func() {
		for {
			select {
			case task := <- wp.tasksQueue:
				wp.tasks <- task
			case <- wp.quit:
				return
			}
		}
	}()
	for range poolSize {
		go func() {
			for {
				select {
				case task, ok := <- wp.tasks:
					if !ok {
						return
					}
					task()
				case <- wp.quit:
					return
				}
			}
		}()
	}
	return wp
}

func (wp *workerPool) wait() {
	wp.wg.Wait()
	close(wp.quit)
}

func (wp *workerPool) submit(task func()) {
	wp.wg.Add(1)
	wp.tasksQueue <- func() {
		defer wp.wg.Done()
		task()
	}
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

	f := &finder{
		finds: make(chan searchItem),
		root: exPath,
		wp: newWorkerPool(100),
	}
	defer f.wp.wait()

	go func() {
		for result := range f.finds {
			fmt.Printf("file: %s, exist in: %s\n", result.name, result.path)
		}
	}()

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
		f.wp.submit(func() {
			f.walkTo(f.root, query)
		})
	}
}

func (f *finder) walkTo(path, query string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			f.wp.submit(func() {
				f.walkTo(filepath.Join(path, entry.Name()), query)
			})
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