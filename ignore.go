// (c) 2011-2012 Alexander Solovyov
// under terms of ISC license

package main

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

func NewIgnorer(wd string) Ignorer {
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
		desc += "\tglobs: " + strings.Join(i.globs, " ") + "\n"
	}

	return desc
}

// read .gitignore and ignore patterns from there
type GitIgnorer struct {
	basepath string
	prefix  string
	f       *os.File
	globs   []string
	dirs    []string
	res     []*regexp.Regexp
}

func NewGitIgnorer(wd string, f *os.File) *GitIgnorer {
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

	globs := []string{}
	dirs := []string{}

	reader := bufio.NewReader(f)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}

		line = bytes.TrimRight(line, " \t")
		if len(line) == 0 {
			continue
		}

		if line[0] == '#' {
			continue
		}

		slashpos := bytes.IndexByte(line, '/')
		switch slashpos {
		case -1:
			globs = append(globs, string(line))
		case len(line) - 1:
			dirs = append(dirs, string(line))
		default:
			// if this is wrong, then blame writers of gitignore manual,
			// it's unobvious as possible
			globs = append(globs, filepath.Join(basepath, string(line)))
		}
	}

	return &GitIgnorer{basepath, prefix, f, globs, dirs, []*regexp.Regexp{}}
}

func (i *GitIgnorer) Ignore(fn string, isdir bool) bool {
	fullpath := filepath.Join(i.basepath, i.prefix, fn)
	prefpath := filepath.Join(i.prefix, fn)
	base := filepath.Base(prefpath)
	dirpath := prefpath[:len(prefpath)-len(base)]

	if isdir && base == ".git" {
		return true
	}

	for _, pat := range i.globs {
		if strings.Index(pat, "/") != -1 {
			if m, _ := filepath.Match(pat, fullpath); m {
				return true
			}
		} else if m, _ := filepath.Match(pat, fn); m {
			return true
		}
	}

	for _, dir := range i.dirs {
		if strings.Contains(dirpath, dir) {
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
	desc := fmt.Sprintf("Ignoring patterns from %s:\n", i.f.Name())
	if len(i.globs) > 0 {
		desc += "\tglobs: "
		for _, x := range i.globs {
			if strings.HasPrefix(x, i.basepath) {
				desc += x[len(i.basepath):] + " "
			} else {
				desc += x + " "
			}
		}
		desc += "\n"
	}

	if len(i.dirs) > 0 {
		desc += "\tdirs: " + strings.Join(i.dirs, " ") + "\n"
	}

	return desc
}
