package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	files, err := ioutil.ReadDir("./")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "reading directory: %v", err)
		return
	}

	fmt.Print("package main\n\n")
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".png" {
			b, err := ioutil.ReadFile(file.Name())
			if err != nil {
				fmt.Printf("reading file: %v", err)
				continue
			}
			fmt.Println(icon(file.Name(), b))
		}
	}
}

func icon(name string, b []byte) string {
	byteString := strings.Replace(fmt.Sprintf("%v", b), " ", ",", -1)
	byteString = byteString[1:(len(byteString) - 1)]

	name = strings.Split(name, ".")[0] //remove .png
	parts := strings.Split(name, "-")
	var varName string
	for _, part := range parts {
		varName += strings.ToUpper(part[:1])
		varName += part[1:]
	}
	return fmt.Sprintf(`var %s = []byte{%s}`, varName, byteString)
}
