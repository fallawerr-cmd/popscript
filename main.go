package main

import (
	"bytes"
	"bufio"
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
		selfPath, err := os.Executable()
		if err == nil {
			self, err := os.ReadFile(selfPath)
			if err == nil {
				marker := []byte("\x00POPSCRIPT_EMBEDDED\x00")
				if bytes.Contains(self, marker) {
					runFile(selfPath)
					return
				}
			}
		}
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

	fmt.Println("Target platform:")
	fmt.Println("  1: Linux")
	fmt.Println("  2: Windows")
	fmt.Print("Choose: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	isWindows := choice == "2"

	fmt.Printf("1/5 Reading %s...\n", scriptPath)
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("      %d bytes\n", len(src))

	fmt.Println("2/5 Reading interpreter binary...")
	selfPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot find interpreter: %v\n", err)
		os.Exit(1)
	}
	self, err := os.ReadFile(selfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read interpreter: %v\n", err)
		os.Exit(1)
	}
	marker := []byte("\x00POPSCRIPT_EMBEDDED\x00")
	if idx := bytes.Index(self, marker); idx != -1 {
		self = self[:idx]
		fmt.Println("      stripped previous embedded script")
	}
	fmt.Printf("      %d bytes\n", len(self))

	fmt.Println("3/5 Scanning for libraries...")
	libs := map[string][]byte{}
	scriptDir := filepath.Dir(scriptPath)

	localEntries, _ := os.ReadDir(scriptDir)
	for _, e := range localEntries {
		if strings.HasSuffix(e.Name(), ".plib") {
			data, err := os.ReadFile(filepath.Join(scriptDir, e.Name()))
			if err == nil {
				libs[e.Name()] = data
				fmt.Printf("      + %s (%d bytes)\n", e.Name(), len(data))
			}
		}
	}

	
	libsDir := filepath.Join(scriptDir, "libs")
	libEntries, _ := os.ReadDir(libsDir)
	for _, e := range libEntries {
		if strings.HasSuffix(e.Name(), ".plib") {
			key := "libs/" + e.Name()
			if _, exists := libs[key]; !exists {
				data, err := os.ReadFile(filepath.Join(libsDir, e.Name()))
				if err == nil {
					libs[key] = data
					fmt.Printf("      + %s (%d bytes)\n", key, len(data))
				}
			}
		}
	}

	if len(libs) == 0 {
		fmt.Println("      no libraries found")
	}

	fmt.Println("4/5 Embedding script and libraries...")
	libMarker := []byte("\x00POPSCRIPT_LIB\x00")
	libSep := []byte("\x00POPSCRIPT_LIBSEP\x00")
	endMarker := []byte("\x00POPSCRIPT_END\x00")

	out := append(self, marker...)
	out = append(out, src...)

	for name, data := range libs {
		out = append(out, libMarker...)
		out = append(out, []byte(name)...)
		out = append(out, libSep...)
		out = append(out, data...)
	}
	out = append(out, endMarker...)

	outName := strings.TrimSuffix(scriptPath, ".pscript")
	if isWindows {
		outName += ".exe"
	}

	fmt.Printf("5/5 Writing %s...\n", outName)
	if err := os.WriteFile(outName, out, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot write output: %v\n", err)
		os.Exit(1)
	}

	info, _ := os.Stat(outName)
	fmt.Printf("Done. Output: %s (%d bytes)\n", outName, info.Size())
	fmt.Printf("Embedded: 1 script + %d libraries\n", len(libs))
	if isWindows {
		fmt.Println("Run on Windows: " + outName)
	} else {
		fmt.Println("Run: ./" + outName)
	}
}

func runFile(path string) {
	var src []byte
	var embeddedLibs map[string][]byte

	marker := []byte("\x00POPSCRIPT_EMBEDDED\x00")
	libMarker := []byte("\x00POPSCRIPT_LIB\x00")
	libSep := []byte("\x00POPSCRIPT_LIBSEP\x00")
	endMarker := []byte("\x00POPSCRIPT_END\x00")

	selfPath, err := os.Executable()
	if err == nil {
		self, err2 := os.ReadFile(selfPath)
		if err2 == nil {
			if idx := bytes.Index(self, marker); idx != -1 {
				rest := self[idx+len(marker):]

				scriptEnd := bytes.Index(rest, libMarker)
				endIdx := bytes.Index(rest, endMarker)

				if scriptEnd != -1 {
					src = rest[:scriptEnd]
				} else if endIdx != -1 {
					src = rest[:endIdx]
				} else {
					src = rest
				}

				embeddedLibs = map[string][]byte{}
				chunk := rest
				for {
					li := bytes.Index(chunk, libMarker)
					if li == -1 {
						break
					}
					chunk = chunk[li+len(libMarker):]
					ei := bytes.Index(chunk, endMarker)
					next := bytes.Index(chunk, libMarker)

					var libChunk []byte
					if next != -1 && (ei == -1 || next < ei) {
						libChunk = chunk[:next]
					} else if ei != -1 {
						libChunk = chunk[:ei]
					} else {
						libChunk = chunk
					}

					si := bytes.Index(libChunk, libSep)
					if si != -1 {
						name := string(libChunk[:si])
						data := libChunk[si+len(libSep):]
						embeddedLibs[name] = data
					}
				}
			}
		}
	}

	if src == nil {
		src, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	if len(embeddedLibs) > 0 {
		tmpDir, err := os.MkdirTemp("", "popscript-*")
		if err == nil {
			defer os.RemoveAll(tmpDir)
			os.MkdirAll(filepath.Join(tmpDir, "libs"), 0755)
			for name, data := range embeddedLibs {
				outPath := filepath.Join(tmpDir, name)
				os.MkdirAll(filepath.Dir(outPath), 0755)
				os.WriteFile(outPath, data, 0644)
			}
			os.Chdir(tmpDir)
		}
	} else {
		dir := filepath.Dir(path)
		if dir != "" && dir != "." {
			os.Chdir(dir)
		}
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
	url := "https://raw.githubusercontent.com/fallawerr-cmd/plibs/main/" + name + ".plib"
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
  pop build <file.pscript>  Build a Popscript file
  pop get <package>         Install a package from the repository
  pop list                  List installed packages`)
}
