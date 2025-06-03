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
	dirF		bool
	checkDist	bool
	maxDistance int
}

type searchItem struct {
	name string
	path string
}

func main() {
	var (
		root = flag.Bool("p", false, "use current position as root")
		isLinux = flag.Bool("l", false, "Is your system Linux")
		checkDir = flag.Bool("dir", false, "Is need to check dir name")
		checkDistance = flag.Bool("d", false, "Check levenshtein distance to word")
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
		semaphore: make(chan struct{}, 100),
		dirF: *checkDir,
		checkDist: *checkDistance,
		maxDistance: 3,
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
		query = strings.ToLower(query)

		f.wg = new(sync.WaitGroup)
		f.finds = make(chan searchItem, 10)

		go func() {
			for result := range f.finds {
				fmt.Printf("finded: %s, exist in: %s\n", result.name, result.path)
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
			if f.dirF {
				if f.shouldMatch(query, entry.Name()) {
					f.finds <- searchItem{
						name: entry.Name(),
						path: path,
					}
				}
			}

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
			if f.shouldMatch(query, entry.Name()) {
				f.finds <- searchItem{
					name: entry.Name(),
					path: path,
				}
			}
		}
	}
}

func (f *finder) shouldMatch(query, name string) bool {
    if f.checkDist {
        return f.levenshteinDistance(query, strings.ToLower(name)) <= f.maxDistance
    }
    return strings.ToLower(name) == query
}

func (f *finder) levenshteinDistance(s1, s2 string) int {
	s1runes := []rune(s1)
	s2runes := []rune(s2)

	l1 := len(s1runes)
	l2 := len(s2runes)

	column := make([]int, l1+1)
	for i := 1; i <= l1; i++ {
		column[i] = i
	}

	for i := 1; i <= l2; i++ {
		column[0] = i
		lastKey := i - 1
		for j := 1; j <= l1; j++ {
			oldKey := column[j]

			k := 0
			if s1runes[j - 1] != s2runes[i - 1] {
				k = 1
			}

			column[j] = min(min(column[j] + 1, column[j - 1] + 1), lastKey + k)
			lastKey = oldKey
		}
	}

	return column[l1]
}