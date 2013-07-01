// (c) 2011-2013 Alexander Solovyov
// under terms of ISC license

package main

import (
	"bytes"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/wsxiaoys/terminal/color"
	"os"
	"path/filepath"
	"regexp"
)

var (
	Author  = "Alexander Solovyov"
	Version = "1.4"

	byteNewLine = []byte("\n")
)

var opts struct {
	IgnoreCase      bool     `short:"i" long:"ignore-case" description:"ignore pattern case"`
	OnlyName        bool     `short:"n" long:"filename" description:"print only filenames"`
	FindFiles       bool     `short:"f" long:"find-files" description:"search for files and not for text in them"`
	IgnoreFiles     []string `short:"x" long:"exclude" description:"exclude files that match the regexp from search" value-name:"RE"`
	AcceptFiles     []string `short:"o" long:"only" description:"search only in files that match the regexp" value-name:"RE"`
	SingleLine      bool     `short:"s" long:"singleline" description:"match on a single line (^/$ will be begginning/end of line)"`
	PlainText       bool     `short:"p" long:"plain" description:"search plain text"`
	Replace         *string  `short:"r" long:"replace" description:"replace found substrings with this string"`
	NoGlobalIgnores bool     `short:"I" long:"no-autoignore" description:"do not read .git/.hgignore files"`
	Force           bool     `long:"force" description:"force replacement in binary files"`
	Verbose         bool     `short:"v" long:"verbose" description:"be verbose (show non-fatal errors, like unreadable files)"`
	ShowVersion     bool     `short:"V" long:"version" description:"show version and exit"`
	ShowHelp        bool     `long:"help" description:"show this help message"`
}

func main() {
	argparser := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash)

	args, err := argparser.Parse()
	if err != nil {
		return
	}

	if opts.ShowVersion {
		fmt.Printf("goreplace %s\n", Version)
		return
	}

	var noIgnores bool
	for _, item := range os.Args[1:] {
		if item == "-I" || item == "--no-autoignore" {
			noIgnores = true
		}
	}

	cwd, _ := os.Getwd()

	ignoreFileMatcher := NewMatcher(cwd, noIgnores)
	ignoreFileMatcher.Append(opts.IgnoreFiles)

	acceptedFileMatcher := NewGeneralMatcher([]string{}, []string{})
	if len(opts.AcceptFiles) > 0 {
		acceptedFileMatcher.Append(opts.AcceptFiles)
	} else {
		acceptedFileMatcher.Append([]string{".*"})
	}

	argparser.Usage = fmt.Sprintf("[OPTIONS] string-to-search\n\n%s",
		ignoreFileMatcher)

	if opts.ShowHelp || len(args) == 0 {
		argparser.WriteHelp(os.Stdout)
		return
	}

	arg := args[0]
	if opts.PlainText {
		arg = regexp.QuoteMeta(arg)
	}
	if opts.IgnoreCase {
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

	filepath.Walk(".", v.Walker(errors))

	select {
	case err := <-errors:
		if opts.Verbose {
			errhandle(err, false, "")
		}
	default:
	}
}

type GRVisitor struct {
	pattern             *regexp.Regexp
	ignoreFileMatcher   Matcher
	acceptedFileMatcher Matcher
	// Used to prevent sparse newline at the end of output
	prependNewLine bool
}

func (v *GRVisitor) Walker(errors chan<- error) filepath.WalkFunc {
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

func (v *GRVisitor) VisitDir(fn string, fi os.FileInfo) bool {
	return !v.ignoreFileMatcher.Match(fn, true)
}

func (v *GRVisitor) VisitFile(fn string, fi os.FileInfo) {
	if fi.Size() == 0 && !opts.FindFiles {
		return
	}

	if v.ignoreFileMatcher.Match(fn, false) {
		return
	}

	if !v.acceptedFileMatcher.Match(fn, false) {
		return
	}

	if opts.FindFiles {
		v.SearchFileName(fn)
		return
	}

	if fi.Size() >= 1024*1024*10 {
		fmt.Fprintf(os.Stderr, "Skipping %s, too big: %d\n", fn, fi.Size())
		return
	}

	f, content := v.GetFileAndContent(fn, fi)
	if f == nil {
		return
	}
	defer f.Close()

	if opts.Replace == nil {
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

	if opts.Replace != nil {
		f, err = os.OpenFile(fn, os.O_RDWR, 0666)
		msg = "can't open file %s for reading and writing"
	} else {
		f, err = os.Open(fn)
		msg = "can't open file %s for reading"
	}

	if err != nil {
		if opts.Verbose {
			errhandle(err, false, msg, fn)
		}
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
			if binary && !opts.OnlyName {
				fmt.Printf("Binary file %s matches\n", fn)
				break
			} else {
				color.Printf("@g%s\n", fn)
			}
		}

		if opts.OnlyName {
			return
		}

		color.Printf("@!@y%d:", info.num)
		colored := v.pattern.ReplaceAllStringFunc(string(info.line),
			func(wrap string) string {
				return color.Sprintf("@Y%s", wrap)
			})
		fmt.Println(colored)
	}

	if len(lines) > 0 {
		v.prependNewLine = true
	}
}

func (v *GRVisitor) SearchFileName(fn string) {
	if !v.pattern.MatchString(fn) {
		return
	}
	colored := v.pattern.ReplaceAllStringFunc(fn,
		func(wrap string) string {
			return color.Sprintf("@Y%s", wrap)
		})
	fmt.Println(colored)
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

	if opts.SingleLine {
		errhandle(
			fmt.Errorf("Can't handle singleline replacements yet"),
			true, "")
	}

	if opts.PlainText {
		errhandle(
			fmt.Errorf("Can't handle plain text replacements yet"),
			true, "")
	}

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	result = v.pattern.ReplaceAllFunc(content, func(s []byte) []byte {
		if binary && !opts.Force {
			errhandle(
				fmt.Errorf("supply --force to force change of binary file"),
				false, "")
		}
		if !changed {
			changed = true
			color.Printf("@g%s", fn)
		}

		changenum += 1
		return []byte(*opts.Replace)
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
	if opts.SingleLine {
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
// newlines on both ends of this slice to contain this slice
func beginend(s []byte, start int, finish int) (begin int, end int) {
	begin = 0
	end = len(s)

	for i := start - 1; i >= 0; i-- {
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

	return begin, end
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
