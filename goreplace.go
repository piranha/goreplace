// (c) 2011-2014 Alexander Solovyov
// under terms of ISC license

package main

import (
	"bytes"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	byten "github.com/pyk/byten"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

const (
	Author  = "Alexander Solovyov"
	Version = "2.4"
)

var byteNewLine = []byte("\n")
var NoColors = false

var opts struct {
	Replace         *string  `short:"r" long:"replace" description:"replace found substrings with RE" value-name:"RE"`
	Force           bool     `short:""  long:"force" description:"force replacement in binary files"`
	IgnoreCase      bool     `short:"i" long:"ignore-case" description:"ignore pattern case"`
	SingleLine      bool     `short:"s" long:"singleline" description:"^/$ will match beginning/end of line"`
	PlainText       bool     `short:"p" long:"plain" description:"treat pattern as plain text"`
	IgnoreFiles     []string `short:"x" long:"exclude" description:"exclude filenames that match regexp RE (multi)" value-name:"RE"`
	AcceptFiles     []string `short:"o" long:"only" description:"search only filenames that match regexp RE (multi)" value-name:"RE"`
	NoGlobalIgnores bool     `short:"I" long:"no-autoignore" description:"do not read .git/.hgignore files"`
	FindFiles       bool     `short:"f" long:"find-files" description:"search in file names"`
	OnlyName        bool     `short:"n" long:"filename" description:"print only filenames"`
	Verbose         bool     `short:"v" long:"verbose" description:"show non-fatal errors (like unreadable files)"`
	NoColors        bool     `short:"c" long:"no-colors" description:"do not show colors in output"`
	NoGroup         bool     `short:"N" long:"no-group" description:"print file name before each line"`
	ShowVersion     bool     `short:"V" long:"version" description:"show version and exit"`
	ShowHelp        bool     `short:"h" long:"help" description:"show this help message"`
}

var argparser = flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash)

func main() {
	args, err := argparser.Parse()
	if err != nil {
		os.Exit(1)
	}

	if opts.ShowVersion {
		fmt.Printf("goreplace %s\n", Version)
		return
	}

	NoColors = opts.NoColors || runtime.GOOS == "windows"

	cwd, _ := os.Getwd()
	ignoreFileMatcher := NewMatcher(cwd, opts.NoGlobalIgnores)
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
	errhandle(err, true)

	if pattern.Match([]byte("")) {
		errhandle(fmt.Errorf("Your pattern matches empty string"), true)
	}

	if opts.Replace != nil {
		s, err := strconv.Unquote(`"` + *opts.Replace + `"`)
		if err != nil {
			errhandle(err, true)
		}
		*opts.Replace = s
	}

	searchFiles(pattern, ignoreFileMatcher, acceptedFileMatcher)
}

func errhandle(err error, exit bool) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s\n", err)
	if exit {
		os.Exit(1)
	}
	return true
}

func searchFiles(pattern *regexp.Regexp, ignoreFileMatcher Matcher,
	acceptedFileMatcher Matcher) {

	printer := &Printer{NoColors, opts.NoGroup, ""}
	v := &GRVisitor{printer, pattern, ignoreFileMatcher, acceptedFileMatcher}

	err := filepath.Walk(".", v.Walk)
	errhandle(err, false)
}

type GRVisitor struct {
	printer             *Printer
	pattern             *regexp.Regexp
	ignoreFileMatcher   Matcher
	acceptedFileMatcher Matcher
	// errors              chan error
}

func (v *GRVisitor) Walk(fn string, fi os.FileInfo, err error) error {
	if err != nil {
		if opts.Verbose {
			errhandle(err, false)
		}
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
		errhandle(fmt.Errorf("Skipping %s, too big: %s\n", fn, byten.Size(fi.Size())),
			false)
		return
	}

	// just skip invalid symlinks
	if fi.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(fn); err != nil {
			if opts.Verbose {
				errhandle(err, false)
			}
			return
		}
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
		if err != nil {
			errhandle(fmt.Errorf("Error writing replacement to file '%s': %s",
				fn, err), true)
		}
		if int64(n) < fi.Size() {
			err := f.Truncate(int64(n))
			if err != nil {
				errhandle(fmt.Errorf("Error truncating file '%s' to size %d",
					f, n), true)
			}
		}
	}
}

func (v *GRVisitor) GetFileAndContent(fn string, fi os.FileInfo) (f *os.File, content []byte) {
	var err error

	if opts.Replace != nil {
		f, err = os.OpenFile(fn, os.O_RDWR, 0666)
	} else {
		f, err = os.Open(fn)
	}

	if err != nil {
		if opts.Verbose {
			errhandle(err, false)
		}
		return
	}

	content = make([]byte, fi.Size())
	n, err := f.Read(content)
	if err != nil {
		errhandle(fmt.Errorf("Error %s", err), false)
		return
	}
	if int64(n) != fi.Size() {
		errhandle(fmt.Errorf("Not whole file '%s' was read, only %d from %d",
			fn, n, fi.Size()), true)
	}

	return
}

func (v *GRVisitor) SearchFile(fn string, content []byte) {
	seen := NewIntSet()
	binary := bytes.IndexByte(content, 0) != -1
	found := v.FindAllIndex(content)
	idxFmt := "%d:"

	if !opts.NoGroup {
		maxVal := 0
		for _, info := range found {
			if info.num > maxVal {
				maxVal = info.num
			}
		}
		idxLength := int(math.Ceil(math.Log10(float64(maxVal))))
		idxFmt = fmt.Sprintf("%%%dd:", idxLength)
	}

	for _, info := range found {
		if !seen.Add(info.num) {
			continue
		}

		if binary && !opts.OnlyName {
			fmt.Printf("Binary file '%s' matches\n", fn)
			return
		}

		if opts.OnlyName {
			v.printer.Printf("@g%s\n", "%s\n", fn)
			return
		}

		colored := v.pattern.ReplaceAllStringFunc(string(info.line),
			func(wrap string) string {
				return v.printer.Sprintf("@Y%s", "%s", wrap)
			})

		v.printer.FilePrintf(fn,
			"@!@y" + idxFmt + "@|%s\n",
			idxFmt + "%s\n",
			info.num,
			colored)
	}
}

func (v *GRVisitor) SearchFileName(fn string) {
	if !v.pattern.MatchString(fn) {
		return
	}
	colored := v.pattern.ReplaceAllStringFunc(fn,
		func(wrap string) string {
			return v.printer.Sprintf("@Y%s", "%s", wrap)
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
	if opts.SingleLine {
		errhandle(fmt.Errorf("Can't handle singleline replacements"),
			true)
	}

	if opts.PlainText {
		errhandle(fmt.Errorf("Can't handle plain text replacements"),
			true)
	}

	changed = false
	changenum := 0
	binary := bytes.IndexByte(content, 0) != -1

	result = v.pattern.ReplaceAllFunc(content, func(s []byte) []byte {
		if binary && !opts.Force {
			errhandle(
				fmt.Errorf("supply --force to force change of binary file"),
				false)
		}
		if !changed {
			changed = true
			v.printer.Printf("@g%s", "%s", fn)
		}

		changenum += 1
		return v.pattern.ReplaceAll(s, []byte(*opts.Replace))
	})

	if changenum > 0 {
		v.printer.Printf("@!@y - %d change%s made\n", " - %d change%s made\n",
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

type IntSet struct {
    set map[int]bool
}

func NewIntSet() *IntSet {
	return &IntSet{make(map[int]bool)}
}

func (set *IntSet) Add(i int) bool {
    _, found := set.set[i]
    set.set[i] = true
    return !found   // False if it existed already
}
