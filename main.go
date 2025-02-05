/*
 * Copyright (c) 2022 Brandon Jordan
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var filePath string
var filename string
var basename string
var contents string
var relativePath string

var included []string

const fileExtension = "cherri"

func main() {
	registerArg("share", "s", "Signing mode. [anyone, contacts] [default=contacts]")
	registerArg("unsigned", "u", "Don't sign compiled Shortcut. Will NOT run on iOS or macOS.")
	registerArg("debug", "d", "Save generated plist. Print debug messages and stack traces.")
	registerArg("output", "o", "Optional output file path. (e.g. /path/to/file.shortcut).")
	if len(os.Args) <= 1 {
		usage()
		os.Exit(0)
	}
	filePath = os.Args[1]
	checkFile(filePath)
	var pathParts = strings.Split(filePath, "/")
	filename = end(pathParts)
	relativePath = strings.Replace(filePath, filename, "", 1)
	var nameParts = strings.Split(filename, ".")
	basename = nameParts[0]
	var bytes, readErr = os.ReadFile(filePath)
	handle(readErr)
	contents = string(bytes)

	if strings.Contains(contents, "#include") {
		parseIncludes()
	}

	if arg("debug") {
		fmt.Printf("Parsing %s... ", filename)
	}
	parse()
	if arg("debug") {
		fmt.Print("\033[32mdone!\033[0m\n")
	}

	if arg("debug") {
		fmt.Println(tokens)
		fmt.Print("\n")
		fmt.Println(variables)
		fmt.Print("\n")
		fmt.Println(menus)
		fmt.Print("\n")
	}

	if arg("debug") {
		fmt.Printf("Generating plist... ")
	}
	var plist = makePlist()
	if arg("debug") {
		fmt.Print("\033[32mdone!\033[0m\n")
	}

	if arg("debug") {
		fmt.Printf("Creating %s.plist... ", basename)
		plistWriteErr := os.WriteFile(basename+".plist", []byte(plist), 0600)
		handle(plistWriteErr)
		fmt.Print("\033[32mdone!\033[0m\n")
	}

	if arg("debug") {
		fmt.Printf("Creating unsigned %s.shortcut... ", basename)
	}
	shortcutWriteErr := os.WriteFile(basename+"_unsigned.shortcut", []byte(plist), 0600)
	handle(shortcutWriteErr)
	if arg("debug") {
		fmt.Print("\033[32mdone!\033[0m\n")
	}

	if !arg("unsigned") {
		sign()
	}
}

func parseIncludes() {
	lines = strings.Split(contents, "\n")
	for l, line := range lines {
		lineIdx = l
		if !strings.Contains(line, "#include") {
			continue
		}
		r := regexp.MustCompile("\"(.*?)\"")
		var includePath = strings.Trim(r.FindString(line), "\"")
		if includePath == "" {
			parserError("No path inside of include")
		}
		if !strings.Contains(includePath, "..") {
			includePath = relativePath + includePath
		}
		if contains(included, includePath) {
			parserError(fmt.Sprintf("File '%s' has already been included.", includePath))
		}
		checkFile(includePath)
		bytes, readErr := os.ReadFile(includePath)
		handle(readErr)
		lines[l] = string(bytes)
		included = append(included, includePath)
	}
	contents = strings.Join(lines, "\n")
	lineIdx = 0
	if strings.Contains(contents, "#include") {
		parseIncludes()
	}
}

func checkFile(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("\n\033[31mFile at path '%s' does not exist!\033[0m\n", filePath)
		os.Exit(1)
	}
	var file, statErr = os.Stat(filePath)
	handle(statErr)
	var nameParts = strings.Split(file.Name(), ".")
	var ext = end(nameParts)
	if ext != fileExtension {
		fmt.Printf("\n\033[31mFile '%s' is not a .%s file!\033[0m\n", filePath, fileExtension)
		os.Exit(1)
	}
}

func sign() {
	var signingMode = "people-who-know-me"
	if arg("share") {
		if argValue("share") == "anyone" {
			signingMode = "anyone"
		}
	}
	if arg("debug") {
		fmt.Printf("Signing %s.shortcut... ", basename)
	}
	var outputPath = basename + ".shortcut"
	if arg("output") {
		outputPath = argValue("output")
	}
	var signBytes, signErr = exec.Command(
		"shortcuts",
		"sign",
		"-i", basename+"_unsigned.shortcut",
		"-o", outputPath,
		"-m", signingMode,
	).Output()
	if signErr != nil {
		if arg("debug") {
			fmt.Print("\033[31mfailed!\033[0m\n")
		}
		fmt.Println("\n\033[31mError: Failed to sign Shortcut, plist may be invalid!\033[0m")
		if len(signBytes) > 0 {
			fmt.Println("shortcuts:", string(signBytes))
		}
		os.Exit(1)
	}
	if arg("debug") {
		fmt.Printf("\033[32mdone!\033[0m\n")
	}
	removeErr := os.Remove(basename + "_unsigned.shortcut")
	handle(removeErr)
}

func end(slice []string) string {
	return slice[len(slice)-1]
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func shortcutsUUID() string {
	return strings.ToUpper(uuid.New().String())
}
