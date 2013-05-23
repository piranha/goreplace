// (c) 2011-2012 Alexander Solovyov
// under terms of ISC license

package main

import (
	"bytes"
	"fmt"
	"github.com/droundy/goopt"
	"github.com/wsxiaoys/terminal/color"
	"os"
	"path/filepath"
	"regexp"
)

var (
	Author  = "Alexander Solovyov"
	Version = "0.4.3"
	Summary = "gr [OPTS] string-to-search\n"

	byteNewLine []byte = []byte("\n")

	ignoreCase = goopt.Flag([]string{"-i", "--ignore-case"}, []string{},
		"ignore pattern case", "")
	onlyName = goopt.Flag([]string{"-n", "--filename"}, []string{},
		"print only filenames", "")
	ignoreFiles = goopt.Strings([]string{"-x", "--exclude"}, "RE",
		"exclude files that match the regexp from search")
	acceptedFiles = goopt.Strings([]string{"-o", "--only"}, "RE",
		"include only files that match this regexp")
	singleline = goopt.Flag([]string{"-s", "--singleline"}, []string{},
		"match on a single line (^/$ will be beginning/end of line)", "")
	plaintext = goopt.Flag([]string{"-p", "--plain"}, []string{},
		"search plain text", "")
	replace = goopt.String([]string{"-r", "--replace"}, "",
		"replace found substrings with this string")
	force = goopt.Flag([]string{"--force"}, []string{},
		"force replacement in binary files", "")
	showVersion = goopt.Flag([]string{"-V", "--version"}, []string{},
		"show version and exit", "")
	noIgnoresGlobal = goopt.Flag([]string{"-I", "--no-autoignore"}, []string{},
		"do not read .git/.hgignore files", "")
	verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{},
		"be verbose (show non-fatal errors, like unreadable files)", "")
)

func main() {
	goopt.Author = Author
	goopt.Version = Version
	goopt.Summary = Summary
	goopt.Usage = func() string {
		return fmt.Sprintf("Usage of goreplace %s:\n\t", Version) +
			goopt.Summary + "\n" + goopt.Help()
	}

	var noIgnores bool
	for _, item := range os.Args[1:] {
		if item == "-I" || item == "--no-autoignore" {
			noIgnores = true
		}
	}

	cwd, _ := os.Getwd()
	ignoreFileMatcher := NewMatcher(cwd, noIgnores)
	acceptedFileMatcher := NewGeneralMatcher([]string{}, []string{})
	goopt.Summary += fmt.Sprintf("\n%s", ignoreFileMatcher)

	goopt.Parse(nil)

	if *showVersion {
		fmt.Printf("goreplace %s\n", goopt.Version)
		return
	}

	ignoreFileMatcher.Append(*ignoreFiles)

	if len(*acceptedFiles) > 0 {
		acceptedFileMatcher.Append(*acceptedFiles)
	} else {
		acceptedFileMatcher.Append([]string{".*"})
	}

	if len(goopt.Args) == 0 {
		println(goopt.Usage())
		return
	}

	arg := goopt.Args[0]
	if *plaintext {
		arg = escapeRegexp(arg)
	}
	if *ignoreCase {
		arg = "(?i:" + arg + ")"
	}

	pattern, err := regexp.Compile(arg)
	errhandle(err, true, "")

	if pattern.Match([]byte("")) {
		errhandle(fmt.Errorf("Your pattern matches empty string"), true, "")
	}

	searchFiles(pattern, ignoreFileMatcher, acceptedFileMatcher)
}

func errhandle(err error, exit bool, moreinfo string, a ...interface{}) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s\n%s\n", err, fmt.Sprintf(moreinfo, a...))
	if exit {
		os.Exit(1)
	}
	return true
}

func searchFiles(pattern *regexp.Regexp, ignoreFileMatcher Matcher,
	acceptedFileMatcher Matcher) {
	v := &GRVisitor{pattern, ignoreFileMatcher, acceptedFileMatcher, false}

	errors := make(chan error, 64)

	filepath.Walk(".", walkFunc(v, errors))

	select {
	case err := <-errors:
		if (*verbose) { errhandle(err, false, "") }
	default:
	}
}

func walkFunc(v *GRVisitor, errors chan<- error) filepath.WalkFunc {
	return func(fn string, fi os.FileInfo, err error) error {
		if err != nil {
			errors <- err
			return nil
		}

		// NOTE: if a directory is a symlink, filepath.Walk won't recurse inside
		if fi.Mode()&os.ModeSymlink != 0 {
			if fi, err = os.Stat(fn); err != nil {
				errors <- err
				return nil
			}
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
	pattern             *regexp.Regexp
	ignoreFileMatcher   Matcher
	acceptedFileMatcher Matcher
	// Used to prevent sparse newline at the end of output
	prependNewLine bool
}

func (v *GRVisitor) VisitDir(fn string, fi os.FileInfo) bool {
	return !v.ignoreFileMatcher.Match(fi.Name(), true)
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

	if v.ignoreFileMatcher.Match(fn, false) {
		return
	}

	if !v.acceptedFileMatcher.Match(fn, false) {
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
		if (*verbose) { errhandle(err, false, msg, fn) }
		return
	}

	content = make([]byte, fi.Size())
	n, err := f.Read(content)
	errhandle(err, true, "can't read file %s", fn)
	if int64(n) != fi.Size() {
		errhandle(fmt.Errorf("Not whole file was read, only %d from %d",
			n, fi.Size()), true, "")
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

		first := len(lines) == 0
		lines = append(lines, info.num)

		if first {
			if binary && !*onlyName {
				fmt.Printf("Binary file %s matches\n", fn)
				break
			} else {
				color.Printf("@g%s\n", fn)
			}
		}

		if *onlyName {
			return
		}

		color.Printf("@!@y%d:", info.num)
		coloredLine := v.pattern.ReplaceAllStringFunc(string(info.line),
			func(wrap string) string {
				return color.Sprintf("@Y%s", wrap)
			})
		fmt.Printf("%s\n", coloredLine)
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
		errhandle(
			fmt.Errorf("Can't handle singleline replacements yet"),
			true, "")
	}

	if *plaintext {
		errhandle(
			fmt.Errorf("Can't handle plain text replacements yet"),
			true, "")
	}

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	result = v.pattern.ReplaceAllFunc(content, func(s []byte) []byte {
		if binary && !*force {
			errhandle(
				fmt.Errorf("supply --force to force change of binary file"),
				false, "")
		}
		if !changed {
			changed = true
			color.Printf("@g%s", fn)
		}

		changenum += 1
		return []byte(*replace)
	})

	if changenum > 0 {
		color.Printf("@!@y - %d change%s made\n",
			changenum, getSuffix(changenum))
	}

	return changed, result
}

type LineInfo struct {
	num  int
	line []byte
}

func (v *GRVisitor) FindAllIndex(content []byte) (res []*LineInfo) {
	if *singleline {
		return v.singlelineFindAllIndex(content)
	}

	linenum, last := 1, 0
	for _, bounds := range v.pattern.FindAllIndex(content, -1) {
		linenum += bytes.Count(content[last:bounds[0]], byteNewLine)
		last = bounds[0]
		begin, end := beginend(content, bounds[0], bounds[1])
		res = append(res, &LineInfo{linenum, content[begin:end]})
	}
	return res
}

func (v *GRVisitor) singlelineFindAllIndex(content []byte) (res []*LineInfo) {
	linenum, begin, end := 1, 0, 0
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

	// -1 to check if current location is the end of a string
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

var toEscape = map[rune]bool{
	'\\': true,
	'.':  true,
	'(':  true,
	')':  true,
	'[':  true,
	']':  true,
	'{':  true,
	'}':  true,
	'+':  true,
	'*':  true,
	'?':  true,
	'|':  true,
	'^':  true,
	'$':  true,
}

func escapeRegexp(arg string) string {
	var buffer bytes.Buffer

	for _, c := range arg {
		if toEscape[c] {
			buffer.WriteRune('\\')
		}
		buffer.WriteRune(c)
	}
	return buffer.String()
}
