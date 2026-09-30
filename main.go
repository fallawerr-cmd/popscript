package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pop/interpreter"
	"pop/lexer"
	"pop/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: pop run <file.pscript>")
			os.Exit(1)
		}
		runFile(os.Args[2])

	case "init":
		fmt.Println("Initialized new PopScript project.")

	case "list":
		listPackages()

	case "build":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: pop build <file.pscript>")
			os.Exit(1)
		}
		buildFile(os.Args[2])

	case "get":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: pop get <package>")
			os.Exit(1)
		}
		downloadPackage(os.Args[2])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func listPackages() {
	entries, err := os.ReadDir("libs")
	if err != nil {
		fmt.Println("No packages installed (libs/ not found)")
		return
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".plib") {
			name := strings.TrimSuffix(e.Name(), ".plib")
			fmt.Printf("  %s\n", name)
			found = true
		}
	}
	if !found {
		fmt.Println("No packages installed")
	}
}

func buildFile(scriptPath string) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	self, err := os.ReadFile(os.Args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading interpreter: %v\n", err)
		os.Exit(1)
	}
	outName := strings.TrimSuffix(scriptPath, ".pscript")
	marker := []byte("\x00POPSCRIPT_EMBEDDED\x00")
	out := append(self, marker...)
	out = append(out, src...)
	if err = os.WriteFile(outName, out, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Built %s\n", outName)
}

func runFile(path string) {
	var src []byte
	self, err := os.ReadFile(os.Args[0])
	marker := []byte("\x00POPSCRIPT_EMBEDDED\x00")
	idx := bytes.Index(self, marker)
	if err == nil && idx != -1 {
		src = self[idx+len(marker):]
	} else {
		src, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		os.Chdir(dir)
	}
	l := lexer.New(string(src))
	tokens, err := l.Tokenize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Lexer error: %v\n", err)
		os.Exit(1)
	}
	p := parser.New(tokens)
	prog, err := p.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}
	interp := interpreter.New()
	if err := interp.Run(prog); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}

func downloadPackage(name string) {
	url := "https://raw.githubusercontent.com/fallawerr/plibs/main/" + name + ".plib"
	fmt.Printf("Fetching %s from fallawerr's repo...\n", name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		fmt.Fprintf(os.Stderr, "Error: package %q not found in repository\n", name)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Error: server returned %d\n", resp.StatusCode)
		os.Exit(1)
	}
	if err := os.MkdirAll("libs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating libs/: %v\n", err)
		os.Exit(1)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
	p := "libs/" + name + ".plib"
	if err := os.WriteFile(p, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Installed %s → %s\n", name, p)
}

func printUsage() {
	fmt.Println(`PopScript interpreter

Usage:
  pop run <file.pscript>    Run a PopScript file
  pop get <package>         Install a package from Pop-Inc repo (not ready)
  pop list                  List installed packages`)
}
