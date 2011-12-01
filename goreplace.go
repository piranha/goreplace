package main

import (
	"bytes"
	"errors"
	"fmt"
	goopt "github.com/droundy/goopt"
	"./highlight"
	"./ignore"
	"os"
	"path/filepath"
	"regexp"
)

var Author = "Alexander Solovyov"
var Version = "0.3.3"
var Summary = "gr [OPTS] string-to-search\n"

var byteNewLine []byte = []byte("\n")

var onlyName = goopt.Flag([]string{"-n", "--filename"}, []string{},
	"print only filenames", "")
var ignoreFiles = goopt.Strings([]string{"-x", "--exclude"}, "RE",
	"exclude files that match the regexp from search")
var singleline = goopt.Flag([]string{"-s", "--singleline"}, []string{},
	"match on a single line (^/$ will be beginning/end of line)", "")
var replace = goopt.String([]string{"-r", "--replace"}, "",
	"replace found substrings with this string")
var force = goopt.Flag([]string{"--force"}, []string{},
	"force replacement in binary files", "")
var showVersion = goopt.Flag([]string{"-v", "--version"}, []string{},
	"show version and exit", "")

func main() {
	goopt.Author = Author
	goopt.Version = Version
	goopt.Summary = Summary
	goopt.Usage = func() string {
		return fmt.Sprintf("Usage of goreplace %s:\n\t", Version) +
			goopt.Summary + "\n" + goopt.Help()
	}

	cwd, _ := os.Getwd()
	ignorer := ignore.New(cwd)
	goopt.Summary += fmt.Sprintf("\n%s", ignorer)

	goopt.Parse(nil)

	if *showVersion {
		println("goreplace " + goopt.Version)
		return
	}

	ignorer.Append(*ignoreFiles)

	if len(goopt.Args) == 0 {
		println(goopt.Usage())
		return
	}

	pattern, err := regexp.Compile(goopt.Args[0])
	errhandle(err, true, "can't compile regexp %s", goopt.Args[0])

	searchFiles(pattern, ignorer)
}

func errhandle(err error, exit bool, moreinfo string, a ...interface{}) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "ERR %s\n%s\n", err, fmt.Sprintf(moreinfo, a...))
	if exit {
		os.Exit(1)
	}
}

func searchFiles(pattern *regexp.Regexp, ignorer ignore.Ignorer) {
	v := &GRVisitor{pattern, ignorer, false}

	errors := make(chan error, 64)

	filepath.Walk(".", walkFunc(v, errors))

	select {
	case err := <-errors:
		errhandle(err, true, "some error")
	default:
	}
}

func walkFunc(v *GRVisitor, errors chan<- error) filepath.WalkFunc {
	return func(fn string, fi os.FileInfo, err error) error {
		if err != nil {
			errors <- err
			return nil
		}

		if fi.IsDir() {
			if !v.VisitDir(fn, fi) {
				return filepath.SkipDir
			}
			return nil
		}

		v.VisitFile(fn, fi)
		return nil
	}
}

type GRVisitor struct {
	pattern *regexp.Regexp
	ignorer ignore.Ignorer
	// Used to prevent sparse newline at the end of output
	prependNewLine bool
}

func (v *GRVisitor) VisitDir(fn string, fi os.FileInfo) bool {
	return !v.ignorer.Ignore(fi.Name(), true)
}

func (v *GRVisitor) VisitFile(fn string, fi os.FileInfo) {
	if fi.IsDir() {
		return
	}

	if fi.Size() >= 1024*1024*10 {
		fmt.Fprintf(os.Stderr, "Skipping %s, too big: %d\n", fn, fi.Size())
		return
	}

	if fi.Size() == 0 {
		return
	}

	if v.ignorer.Ignore(fn, false) {
		return
	}

	f, content := v.GetFileAndContent(fn, fi)
	defer f.Close()

	if len(*replace) == 0 {
		v.SearchFile(fn, content)
		return
	}

	changed, result := v.ReplaceInFile(fn, content)
	if changed {
		f.Seek(0, 0)
		n, err := f.Write(result)
		errhandle(err, true, "Error writing replacement in file %s", fn)
		if int64(n) < fi.Size() {
			err := f.Truncate(int64(n))
			errhandle(err, true, "Error truncating file to size %d", f)
		}
	}
}

func (v *GRVisitor) GetFileAndContent(fn string, fi os.FileInfo) (f *os.File, content []byte) {
	var err error
	var msg string

	if len(*replace) > 0 {
		f, err = os.OpenFile(fn, os.O_RDWR, 0666)
		msg = "can't open file %s for reading and writing"
	} else {
		f, err = os.Open(fn)
		msg = "can't open file %s for reading"
	}

	if err != nil {
		errhandle(err, false, msg, fn)
		return
	}

	content = make([]byte, fi.Size())
	n, err := f.Read(content)
	errhandle(err, true, "can't read file %s", fn)
	if int64(n) != fi.Size() {
		panic(fmt.Sprintf("Not whole file was read, only %d from %d",
			n, fi.Size()))
	}

	return
}

func (v *GRVisitor) SearchFile(fn string, content []byte) {
	lines := IntList([]int{})
	binary := false

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	for _, info := range v.FindAllIndex(content) {
		if lines.Contains(info.num) {
			continue
		}

		if v.prependNewLine {
			fmt.Println("")
			v.prependNewLine = false
		}

		var first = len(lines) == 0
		lines = append(lines, info.num)

		if first {
			if binary && !*onlyName {
				fmt.Printf("Binary file %s matches\n", fn)
				break
			} else {
				highlight.Printf("green", "%s\n", fn)
			}
		}

		if *onlyName {
			return
		}

		highlight.Printf("bold yellow", "%d:", info.num)
		highlight.Reprintlnf("on_yellow", v.pattern, "%s", info.line)
	}

	if len(lines) > 0 {
		v.prependNewLine = true
	}
}

func getSuffix(num int) string {
	if num > 1 {
		return "s"
	}
	return ""
}

func (v *GRVisitor) ReplaceInFile(fn string, content []byte) (changed bool, result []byte) {
	changed = false
	binary := false
	changenum := 0

	if *singleline {
		panic("Can't handle singleline replacements yet")
	}

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	result = v.pattern.ReplaceAllFunc(content, func(s []byte) []byte {
		if binary && !*force {
			errhandle(
				errors.New("supply --force to force change of binary file"),
				false, "")
		}
		if !changed {
			changed = true
			highlight.Printf("green", "%s", fn)
		}

		changenum += 1
		return []byte(*replace)
	})

	if changenum > 0 {
		highlight.Printf("bold yellow", " - %d change%s made\n",
			changenum, getSuffix(changenum))
	}

	return changed, result
}

type LineInfo struct {
	num  int
	line []byte
}

// will return slice of [linenum, line] slices
func (v *GRVisitor) FindAllIndex(content []byte) (res []*LineInfo) {
	linenum := 1

	if *singleline {
		begin, end := 0, 0
		for i := 0; i < len(content); i++ {
			if content[i] == '\n' {
				end = i
				line := content[begin:end]
				if v.pattern.Match(line) {
					res = append(res, &LineInfo{linenum, line})
				}
				linenum += 1
				begin = end + 1
			}
		}
		return res
	}

	last := 0
	for _, bounds := range v.pattern.FindAllIndex(content, -1) {
		linenum += bytes.Count(content[last:bounds[0]], byteNewLine)
		last = bounds[0]
		begin, end := beginend(content, bounds[0], bounds[1])
		res = append(res, &LineInfo{linenum, content[begin:end]})
	}
	return res
}

// Given a []byte, start and finish of some inner slice, will find nearest
// newlines on both ends of this slice
func beginend(s []byte, start int, finish int) (begin int, end int) {
	begin = 0
	end = len(s)

	for i := start; i >= 0; i-- {
		if s[i] == byteNewLine[0] {
			begin = i + 1
			break
		}
	}

	// -1 to check if current location is not end of string
	for i := finish - 1; i < len(s); i++ {
		if s[i] == byteNewLine[0] {
			end = i
			break
		}
	}

	return
}

type IntList []int

func (il IntList) Contains(i int) bool {
	for _, x := range il {
		if x == i {
			return true
		}
	}
	return false
}
