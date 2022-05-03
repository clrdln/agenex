package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

func parse(filename string) map[string]string {
	out := make(map[string]string)
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) <= 1 || fields[0][0] == '#' {
			continue
		}
		mimeType := fields[0]
		for _, ext := range fields[1:] {
			if ext[0] == '#' {
				break
			}
			out["."+ext] = mimeType
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return out
}

func main() {
	f := os.Args[1]
	out := parse(f)
	fmt.Printf("package main\n")
	fmt.Printf("\n")
	fmt.Printf("\n")
	fmt.Printf("var MimeDict = map[string]string {\n")

	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("\t%q: %q,\n", k, out[k])
	}
	fmt.Printf("}\n")
}
