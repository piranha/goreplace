// (c) 2011-2013 Alexander Solovyov
// under terms of ISC license

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"github.com/piranha/goreplace/fnmatch"
)

type Matcher interface {
	Match(fn string, isdir bool) bool
	Append(pats []string)
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func NewMatcher(wd string, noIgnores bool) Matcher {
	path := wd
	if !filepath.IsAbs(path) {
		panic("Given path should be absolute")
	}

	for !noIgnores {
		if filepath.Dir(path) == path {
			break
		}

		if dirExists(filepath.Join(path, ".hg")) {
			return NewHgMatcher(wd, filepath.Join(path, ".hgignore"))
		}

		if dirExists(filepath.Join(path, ".git")) {
			return NewGitMatcher(wd, filepath.Join(path, ".gitignore"))
		}

		// f, err = os.Open(filepath.Join(path, ".git"))
		// if err == nil {
		// 	return NewGitMatcher(wd, f)
		// }

		path = filepath.Clean(filepath.Join(path, ".."))
	}

	return NewGeneralMatcher(generalDirs, generalPats)
}

// Ignore common patterns
type GeneralMatcher struct {
	dirs []string
	res  []*regexp.Regexp
	both []*regexp.Regexp
}

var generalDirs = []string{"autom4te.cache", "blib", "_build", ".bzr", ".cdv",
	"cover_db", "CVS", "_darcs", "~.dep", "~.dot", ".git", ".hg", "~.nib",
	".pc", "~.plst", "RCS", "SCCS", "_sgbak", ".svn", "_obj"}
var generalPats = []string{`~$`, `#.+#$`, `[._].*\.swp$`,
	`core\.[0-9]+$`, `\.pyc$`, `\.o$`, `\.6$`}

func NewGeneralMatcher(dirs []string, filePats []string) *GeneralMatcher {
	res := make([]*regexp.Regexp, len(filePats))
	for i, pat := range filePats {
		res[i] = regexp.MustCompile(pat)
	}
	return &GeneralMatcher{dirs, res, []*regexp.Regexp{}}
}

func (i *GeneralMatcher) Match(fn string, isdir bool) bool {
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

func (i *GeneralMatcher) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s\n", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *GeneralMatcher) String() string {
	return "General ignorer"
}

// read .hgignore and ignore patterns from there
type HgMatcher struct {
	prefix string
	fp     string
	res    []*regexp.Regexp
	globs  []string
}

var hgSyntaxes = map[string]bool{
	"re":     true,
	"regexp": true,
	"glob":   false,
}

func NewHgMatcher(wd string, fp string) *HgMatcher {
	var prefix string
	basepath := filepath.Clean(filepath.Join(fp, ".."))
	if strings.HasPrefix(wd, basepath) {
		prefix = wd[len(basepath):]
		if len(prefix) > 0 && prefix[0] == '/' {
			prefix = prefix[1:]
		}
	} else {
		prefix = ""
	}

	res := []*regexp.Regexp{}
	globs := []string{}

	f, err := os.Open(fp)
	if err != nil {
		return &HgMatcher{prefix, fp, res, globs}
	}

	reader := bufio.NewReader(f)
	isRe := true
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
	return &HgMatcher{prefix, fp, res, globs}
}

func (i *HgMatcher) Match(fn string, isdir bool) bool {
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

func (i *HgMatcher) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *HgMatcher) String() string {
	desc := fmt.Sprintf("Ignoring patterns from %s:\n", i.fp)
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
type GitMatcher struct {
	basepath string
	prefix   string
	fp       string
	globs    []string
	dirs     []string
	res      []*regexp.Regexp
}

func NewGitMatcher(wd string, fp string) *GitMatcher {
	var prefix string
	basepath := filepath.Clean(filepath.Join(fp, ".."))
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

	f, err := os.Open(fp)
	if err != nil {
		return &GitMatcher{basepath, prefix, fp, globs, dirs, []*regexp.Regexp{}}
	}

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

	return &GitMatcher{basepath, prefix, fp, globs, dirs, []*regexp.Regexp{}}
}

func (i *GitMatcher) Match(fn string, isdir bool) bool {
	fullpath := filepath.Join(i.basepath, i.prefix, fn)
	prefpath := filepath.Join(i.prefix, fn)
	base := filepath.Base(prefpath)
	dirpath := prefpath[:len(prefpath)-len(base)]

	if isdir && base == ".git" {
		return true
	}

	for _, pat := range i.globs {
		if strings.Index(pat, "/") != -1 {
			if m, _ := fnmatch.Match(pat, fullpath); m {
				return true
			}
		} else if m, _ := filepath.Match(pat, fn); m {
			return true
		}
	}

	for _, pat := range i.res {
		if pat.Match([]byte(fn)) {
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

func (i *GitMatcher) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if errhandle(err, false, "can't compile pattern %s", pat) {
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *GitMatcher) String() string {
	desc := fmt.Sprintf("Ignoring patterns from %s:\n", i.fp)
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

	if len(i.res) > 0 {
		desc += "\tregular expressions: "
		for _, x := range i.res {
			desc += x.String() + " "
		}
		desc += "\n"
	}

	if len(i.dirs) > 0 {
		desc += "\tdirs: " + strings.Join(i.dirs, " ") + "\n"
	}

	return desc
}
