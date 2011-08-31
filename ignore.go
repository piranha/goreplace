package ignore

import (
	"os"
	"path/filepath"
	"bufio"
	"fmt"
	"regexp"
	"bytes"
	"strings"
)

type Ignorer interface {
	Ignore(fn string, isdir bool) bool
	Append(pats []string)
}

func errhandle(err os.Error, exit bool, moreinfo string, a ...interface{}) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "ERR %s\n%s\n", err,
		fmt.Sprintf(moreinfo, a...))
	if exit {
		os.Exit(1)
	}
	return true
}

func New(wd string) Ignorer {
	path := wd
	if path[0] != '/' {
		panic("Given path should be anchored at /")
	}

	for {
		if path == "/" {
			break
		}

		f, err := os.Open(filepath.Join(path, ".hgignore"))
		if err == nil {
			return NewHgIgnorer(wd, f)
		}

		f, err = os.Open(filepath.Join(path, ".gitignore"))
		if err == nil {
			return NewGitIgnorer(wd, f)
		}

		path = filepath.Clean(filepath.Join(path, ".."))
	}

	return NewGeneralIgnorer()
}

// Ignore common patterns
type GeneralIgnorer struct {
	dirs []string
	res  []*regexp.Regexp
	both []*regexp.Regexp
}

var generalDirs = []string{"autom4te.cache", "blib", "_build", ".bzr", ".cdv",
	"cover_db", "CVS", "_darcs", "~.dep", "~.dot", ".git", ".hg", "~.nib",
	".pc", "~.plst", "RCS", "SCCS", "_sgbak", ".svn", "_obj"}
var generalPats = []string{`~$`, `#.+#$`, `[._].*\.swp$`,
	`core\.[0-9]+$`, `\.pyc$`, `\.o$`, `\.6$`}

func NewGeneralIgnorer() *GeneralIgnorer {
	res := make([]*regexp.Regexp, len(generalPats))
	for i, pat := range generalPats {
		res[i] = regexp.MustCompile(pat)
	}
	return &GeneralIgnorer{generalDirs, res, []*regexp.Regexp{}}
}

func (i *GeneralIgnorer) Ignore(fn string, isdir bool) bool {
	if isdir {
		base := filepath.Base(fn)
		for _, x := range i.dirs {
			if base == x {
				return true
			}
		}
	}

	for _, x := range i.res {
		if x.Match([]byte(fn)) {
			return true
		}
	}

	return false
}

func (i *GeneralIgnorer) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s\n", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *GeneralIgnorer) String() string {
	return "General ignorer"
}

// read .hgignore and ignore patterns from there
type HgIgnorer struct {
	prefix string
	f      *os.File
	res    []*regexp.Regexp
	globs  []string
}

var hgSyntaxes = map[string]bool{
	"re":     true,
	"regexp": true,
	"glob":   false,
}

func NewHgIgnorer(wd string, f *os.File) *HgIgnorer {
	reader := bufio.NewReader(f)
	isRe := true

	res := []*regexp.Regexp{}
	globs := []string{}

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		// strip comments
		comment := bytes.IndexByte(line, '#')
		switch comment {
		case 0:
			continue
		case -1:
		default:
			line = line[:comment]
		}

		line = bytes.TrimRight(line, " \t")
		if len(line) == 0 {
			continue
		}

		// if it's a syntax changer
		if bytes.HasPrefix(line, []byte("syntax:")) {
			s := bytes.TrimSpace(line[7:])
			if isre, ok := hgSyntaxes[string(s)]; ok {
				isRe = isre
			}
			continue
		}

		// actually append line
		pat := string(line)
		if isRe {
			re, err := regexp.Compile(pat)
			if errhandle(err, false, "can't compile pattern %s", pat) {
				continue
			}
			res = append(res, re)
		} else {
			globs = append(globs, pat)
		}
	}

	var prefix string
	basepath := filepath.Clean(filepath.Join(f.Name(), ".."))
	if strings.HasPrefix(wd, basepath) {
		prefix = wd[len(basepath):]
		if len(prefix) > 0 && prefix[0] == '/' {
			prefix = prefix[1:]
		}
	} else {
		prefix = ""
	}
	return &HgIgnorer{prefix, f, res, globs}
}

func (i *HgIgnorer) Ignore(fn string, isdir bool) bool {
	if len(i.prefix) > 0 {
		fn = filepath.Join(i.prefix, fn)
	}
	base := filepath.Base(fn)

	if isdir && base == ".hg" {
		return true
	}

	for _, x := range i.res {
		if x.Match([]byte(fn)) {
			return true
		}
	}

	for _, x := range i.globs {
		if m, _ := filepath.Match(x, base); m {
			return true
		}
	}

	return false
}

func (i *HgIgnorer) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *HgIgnorer) String() string {
	desc := fmt.Sprintf("Ignoring patterns from %s:\n", i.f.Name())
	if len(i.res) > 0 {
		desc += "\tregular expressions: "
		for _, x := range i.res {
			desc += x.String() + " "
		}
		desc += "\n"
	}

	if len(i.globs) > 0 {
		desc += "\tglobs: "
		for _, x := range i.globs {
			desc += x + " "
		}
		desc += "\n"
	}

	return desc
}

// read .gitignore and ignore patterns from there
type GitIgnorer struct {
	prefix  string
	f       *os.File
	entries []string
	res     []*regexp.Regexp
}

func NewGitIgnorer(wd string, f *os.File) *GitIgnorer {
	// reader := bufio.NewReader(f)

	var prefix string
	basepath := filepath.Clean(filepath.Join(f.Name(), ".."))
	if strings.HasPrefix(wd, basepath) {
		prefix = wd[len(basepath):]
		if len(prefix) > 0 && prefix[0] == '/' {
			prefix = prefix[1:]
		}
	} else {
		prefix = ""
	}

	return &GitIgnorer{prefix, f, []string{}, []*regexp.Regexp{}}
}

func (i *GitIgnorer) Ignore(fn string, isdir bool) bool {
	if len(i.prefix) > 0 {
		fn = filepath.Join(i.prefix, fn)
	}

	for _, x := range i.res {
		if x.Match([]byte(fn)) {
			return true
		}
	}

	return false
}

func (i *GitIgnorer) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *GitIgnorer) String() string {
	return fmt.Sprintf("Ignoring patterns from %s", i.f.Name())
}
