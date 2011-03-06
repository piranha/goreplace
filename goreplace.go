package main

import (
	"os"
	"path"
	"fmt"
	"regexp"
	"bytes"
	goopt "github.com/droundy/goopt"
	"./highlight"
)

var Author = "Alexander Solovyov"
var Version = "0.2"
var Summary = "gr [OPTS] string-to-search\n"
var Description = `Go search and replace (not done yet) in files`

var byteNewLine []byte = []byte("\n")
// Used to prevent appear of sparse newline at the end of output
var prependNewLine = false


type StringList []string

var IgnoreDirs = StringList{"autom4te.cache", "blib", "_build", ".bzr", ".cdv",
	"cover_db", "CVS", "_darcs", "~.dep", "~.dot", ".git", ".hg", "~.nib",
	".pc", "~.plst", "RCS", "SCCS", "_sgbak", ".svn"}

type RegexpList []*regexp.Regexp

var IgnoreFiles = regexpList([]string{`~$`, `#.+#$`, `[._].*\.swp$`, `core\.[0-9]+$`,
	`\.pyc$`, `\.o$`, `\.6$`})


var onlyName = goopt.Flag([]string{"-n", "--filename"}, []string{},
	"print only filenames", "")
var ignoreFiles = goopt.Strings([]string{"-x", "--exclude"}, "RE",
	"exclude files that match the regexp from search")
var singleline = goopt.Flag([]string{"-s", "--singleline"}, []string{},
	"match on a single line (^/$ will be beginning/end of line)", "")
var replace = goopt.String([]string{"-r", "--replace"}, "",
	"replace found substrings with this string")
var force = goopt.Flag([]string{"-f", "--force"}, []string{},
	"force replacement in binary files", "")

func main() {
	goopt.Author = Author
	goopt.Version = Version
	goopt.Summary = Summary
	goopt.Description = func() string { return Description }
	goopt.Parse(nil)

	if len(goopt.Args) == 0 {
		println(goopt.Usage())
		return
	}

	IgnoreFiles = append(IgnoreFiles, regexpList(*ignoreFiles)...)

	pattern, err := regexp.Compile(goopt.Args[0])
	errhandle(err, "can't compile regexp %s", goopt.Args[0])

	searchFiles(pattern)
}

func errhandle(err os.Error, moreinfo string, a ...interface{}) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "ERR %s\n%s\n", err,
		fmt.Sprintf(moreinfo, a...))
	os.Exit(1)
}

func regexpList(sa []string) RegexpList {
	ra := make(RegexpList, len(sa))
	for i, s := range sa {
		ra[i] = regexp.MustCompile(s)
	}
	return ra
}

func searchFiles(pattern *regexp.Regexp) {
	v := &GRVisitor{pattern}

	errors := make(chan os.Error, 64)

	path.Walk(".", v, errors)

	select {
	case err := <-errors:
		errhandle(err, "some error")
	default:
	}
}

type GRVisitor struct {
	pattern *regexp.Regexp
}

func (v *GRVisitor) VisitDir(fn string, fi *os.FileInfo) bool {
	if IgnoreDirs.Contains(fi.Name) {
		return false
	}
	return true
}

func (v *GRVisitor) VisitFile(fn string, fi *os.FileInfo) {
	if IgnoreFiles.Match(fn) {
		return
	}

	if fi.Size >= 1024*1024*10 {
		fmt.Fprintf(os.Stderr, "Skipping %s, too big: %d\n", fn, fi.Size)
		return
	}

	if fi.Size == 0 {
		return
	}

	f, content := v.GetFileAndContent(fn, fi)

	if len(*replace) > 0 {
		changed, result := v.ReplaceInFile(fn, content)
		if changed {
			f.Seek(0, 0)
			n, err := f.Write(result)
			errhandle(err, "Error writing replacement in file %s", fn)
			if int64(n) > fi.Size {
				err := f.Truncate(int64(n))
				errhandle(err, "Error truncating file to size %d", f)
			}
		}
	} else {
		v.SearchFile(fn, content)
	}

	f.Close()
}

func (v *GRVisitor) GetFileAndContent(fn string, fi *os.FileInfo) (f *os.File, content []byte) {
	var err os.Error
	if len(*replace) > 0 {
		f, err = os.Open(fn, os.O_RDWR, 0666)
		errhandle(err, "can't open file %s for reading and writing", fn)
	} else {
		f, err = os.Open(fn, os.O_RDONLY, 0666)
		errhandle(err, "can't open file %s for reading", fn)
	}

	content = make([]byte, fi.Size)
	n, err := f.Read(content)
	errhandle(err, "can't read file %s", fn)
	if int64(n) != fi.Size {
		panic(fmt.Sprintf("Not whole file was read, only %d from %d",
			n, fi.Size))
	}

	return
}


func (v *GRVisitor) SearchFile(fn string, content []byte) {
	hadOutput := false
	binary := false

	if bytes.IndexByte(content, 0) != -1 {
		binary = true
	}

	for _, info := range v.FindAllIndex(content) {
		if prependNewLine {
			fmt.Println("")
			prependNewLine = false
		}

		if !hadOutput {
			hadOutput = true
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

	if hadOutput {
		prependNewLine = true
	}
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

	result = v.pattern.ReplaceAllFunc(content, func (s []byte) []byte {
		if binary && !*force {
			errhandle(
				os.NewError("supply --force to force change of binary file"),
				"")
		}
		if !changed {
			changed = true
			highlight.Printf("green", "%s", fn)
		}

		changenum += 1
		return []byte(*replace)
	})

	highlight.Printf("bold yellow", " - %d changes made\n", changenum)

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


func (sl StringList) Contains(s string) bool {
	for _, x := range sl {
		if x == s {
			return true
		}
	}
	return false
}

func (rl RegexpList) Match(s string) bool {
	for _, x := range rl {
		if x.Match([]byte(s)) {
			return true
		}
	}
	return false
}
