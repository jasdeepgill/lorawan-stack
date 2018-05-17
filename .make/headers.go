// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//+build ignore

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	makeRegex      = regexp.MustCompile(".*\\.make$")
	makefileRegex  = regexp.MustCompile(".*Makefile$")
	shRegex        = regexp.MustCompile(".*\\.sh$")
	generatedRegex = regexp.MustCompile("generated")
)

func prefixFunction(filename string) func(string) string {
	byteFilename := []byte(filename)
	commentPrefix := "//"
	if makeRegex.Match(byteFilename) || makefileRegex.Match(byteFilename) || shRegex.Match(byteFilename) {
		commentPrefix = "#"
	}
	return func(line string) string {
		if line == "" || line == "\n" {
			return fmt.Sprintf("%s%s", commentPrefix, line)
		}
		return fmt.Sprintf("%s %s", commentPrefix, line)
	}
}

func hasHeaders(licenseContent []byte, filename string) (valid, generated bool, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	withPrefix := prefixFunction(filename)

	checkedFileReader := bufio.NewReader(file)
	licenseReader := bufio.NewReader(bytes.NewBuffer(licenseContent))
	for {
		licenseLine, err := licenseReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, false, fmt.Errorf("could not read license content from bytes buffer (%s)", err)
		} else if err == io.EOF {
			return true, false, nil
		}
		checkedFileLine, err := checkedFileReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, false, fmt.Errorf("could not read file (%s)", err)
		}

		expected := withPrefix(licenseLine)
		if checkedFileLine != expected {
			if generatedRegex.Match([]byte(checkedFileLine)) {
				return false, true, nil
			}
			return false, false, nil
		}
	}
}

func addHeader(licenseContent []byte, filename string) error {
	tempFilename := fmt.Sprintf("%s-fixed-headers", filename)
	newFile, err := os.Create(tempFilename)
	if err != nil {
		return err
	}

	withPrefix := prefixFunction(filename)

	licenseReader := bufio.NewReader(bytes.NewBuffer(licenseContent))
	for {
		licenseLine, err := licenseReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("could not read license content from bytes buffer (%s)", err)
		} else if err == io.EOF {
			break
		}

		if _, err := newFile.WriteString(withPrefix(licenseLine)); err != nil {
			return err
		}
	}

	if _, err := newFile.Write([]byte("\n")); err != nil {
		return err
	}

	originalFile, err := os.Open(filename)
	if err != nil {
		return err
	}

	if _, err := io.Copy(newFile, originalFile); err != nil {
		return err
	}

	if err := originalFile.Close(); err != nil {
		return err
	}

	if err := newFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempFilename, filename)
}

func nbLines(str []byte) int {
	nb := 1
	for _, char := range str {
		if char == '\n' {
			nb++
		}
	}
	return nb
}

func removeHeaders(nbLines int, filename string) error {
	tempFilename := fmt.Sprintf("%s-without-headers", filename)
	newFile, err := os.Create(tempFilename)
	if err != nil {
		return err
	}

	originalFile, err := os.Open(filename)
	if err != nil {
		return err
	}

	originalFileReader := bufio.NewReader(originalFile)
	for line := 0; line < nbLines; line++ {
		_, err := originalFileReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("could not read original file content (%s)", err)
		}
	}

	if _, err := io.Copy(newFile, originalFileReader); err != nil {
		return err
	}

	if err := originalFile.Close(); err != nil {
		return err
	}

	if err := newFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempFilename, filename)
}

type headersOperation struct {
	licenseContent []byte
	filenames      []string
}

func (o headersOperation) check() bool {
	allFilesValid := true
	for _, file := range o.filenames {
		if valid, generated, err := hasHeaders(o.licenseContent, file); err != nil {
			log.Printf("Could not check headers in %s: %s\n", file, err)
			allFilesValid = false
		} else if !valid && !generated {
			log.Printf("Invalid headers in %s.\n", file)
			allFilesValid = false
		}
	}
	return allFilesValid
}

func (o headersOperation) remove() bool {
	var wasError error
	for _, file := range o.filenames {
		if valid, generated, err := hasHeaders(o.licenseContent, file); err != nil {
			log.Printf("Could not check headers in %s: %s\n", file, err)
			wasError = err
		} else if !generated {
			if !valid {
				log.Printf("No headers in %s.\n", file)
			} else {
				if err := removeHeaders(nbLines(o.licenseContent), file); err != nil {
					log.Printf("Could not remove headers in %s: %s\n", file, err)
					wasError = err
				}
			}
		}
	}
	return wasError == nil
}

func (o headersOperation) fix() bool {
	var wasError error
	for _, file := range o.filenames {
		if valid, generated, err := hasHeaders(o.licenseContent, file); err != nil {
			log.Printf("Could not remove headers in %s: %s\n", file, err)
			wasError = err
		} else if !valid && !generated {
			if err := addHeader(o.licenseContent, file); err != nil {
				log.Printf("Could not fix %s: %s\n", file, err)
			} else {
				log.Printf("Fixed headers in %s.\n", file)
			}
		}
	}
	return wasError == nil
}

func executeOperation(command, licenseFilePath string, files []string) (success bool) {
	licenseContent, err := ioutil.ReadFile(licenseFilePath)
	if err != nil {
		log.Fatalf("Could not read license content in %s: %s\n", licenseFilePath, err)
	}

	operation := headersOperation{
		filenames:      files,
		licenseContent: licenseContent,
	}

	switch command {
	case "remove":
		success = operation.remove()
	case "fix":
		success = operation.fix()
	case "check":
		success = operation.check()
	default:
		log.Printf("Unknown command %s.\n", command)
	}
	return
}

func main() {
	files := []string{}
	if filenames := os.Getenv("FILES"); filenames != "" {
		files = strings.Split(filenames, "\n")
	}
	if len(os.Args) <= 1 {
		fmt.Println("Usage: headers.go {check,remove,fix} [... files to process] [LICENSE_HEADER_PATH=<path to the file containing the header>]")
		os.Exit(1)
	}

	command := os.Args[1]
	if len(files) == 0 && len(os.Args) >= 3 {
		files = os.Args[2:]
	}

	licenseFilePath := os.Getenv("HEADER_FILE")
	if licenseFilePath == "" {
		licenseFilePath = ".make/header.txt"
	}

	successful := executeOperation(command, licenseFilePath, files)
	if !successful {
		os.Exit(1)
	}
}
