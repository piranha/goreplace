// (c) 2011-2014 Alexander Solovyov
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
		if filepath.Dir(path) == path { // top directory
			break
		}

		if dirExists(filepath.Join(path, ".hg")) {
			return NewHgMatcher(wd, filepath.Join(path, ".hgignore"))
		}

		if dirExists(filepath.Join(path, ".git")) {
			return NewGitMatcher(wd, filepath.Join(path, ".gitignore"))
		}

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
		if err != nil {
			errhandle(fmt.Errorf("can't compile pattern %s\n", pat), false)
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
			if err != nil {
				errhandle(fmt.Errorf("can't compile pattern %s\n", pat), false)
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
		if err != nil {
			errhandle(fmt.Errorf("can't compile pattern %s\n", pat), false)
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *HgMatcher) String() string {
	desc := fmt.Sprintf("Ignoring patterns from %s:", i.fp)
	if len(i.res) > 0 {
		desc += "\n\tregular expressions: "
		for _, x := range i.res {
			desc += x.String() + " "
		}
	}

	if len(i.globs) > 0 {
		desc += "\n\tglobs: " + strings.Join(i.globs, " ")
	}

	return desc
}

// read .gitignore and ignore patterns from there
type GitMatcher struct {
	basepath string
	prefix   string
	fp       string
	globs    []string // will be used for showing help only
	globres  []*regexp.Regexp
	res      []*regexp.Regexp
}

// many thanks to Steve Losh for this algorithm
// https://github.com/sjl/friendly-find/blob/master/ffind#L167-216
func gitGlobRe(s string) *regexp.Regexp {
	var pat bytes.Buffer
	if strings.Contains(s, "/") {
		// Patterns with a slash have to match against the entire pathname, so
		// they need to be rooted at the beginning
		pat.WriteString("^./")
	} else {
		// Patterns without a slash match against basename, which is simulated
		// by including last path divider in the pattern
		pat.WriteString("/")
	}
	s = strings.TrimLeft(s, "/")

	i := 0
	n := len(s)

	for i < n {
		c := s[i]
		i += 1

		switch c {
		case '?':
			pat.WriteByte('.')
		case '*':
			if i == n {
				pat.WriteString(".*")
			} else {
				pat.WriteString("[^/]*")
			}
		case '[':
			j := i
			if j < n && (s[j] == '!' || s[j] == ']') {
				j += 1
			}
			for j < n && s[j] != ']' {
				j += 1
			}

			if j >= n {
				pat.WriteString("\\[")
			} else {
				stuff := strings.Replace(s[i:j], "\\", "\\\\", -1)
				i = j + 1
				if stuff[0] == '!' {
					stuff = "^" + stuff[1:]
				} else if stuff[0] == '^' {
					stuff = "\\" + stuff
				}
				pat.WriteString("[")
				pat.WriteString(stuff)
				pat.WriteString("]")
			}
		default:
			pat.WriteString(regexp.QuoteMeta(string(c)))
		}

		if i == n && c != '/' {
			pat.WriteByte('$')
		}
	}

	re, err := regexp.Compile(pat.String())
	if err != nil {
		errhandle(fmt.Errorf("can't parse pattern '%s': %s", s, err), false)
	}
	return re
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
	globres := []*regexp.Regexp{}

	f, err := os.Open(fp)
	if err != nil {
		return &GitMatcher{basepath, prefix, fp, globs, globres, []*regexp.Regexp{}}
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

		globs = append(globs, string(line))
		globres = append(globres, gitGlobRe(string(line)))
	}

	return &GitMatcher{basepath, prefix, fp, globs, globres, []*regexp.Regexp{}}
}

func (i *GitMatcher) Match(fn string, isdir bool) bool {
	// no point in ignore whole current directory
	if fn == "." {
		return false
	}

	path := fmt.Sprintf(".%c", filepath.Separator) + filepath.Join(i.prefix, fn)
	base := filepath.Base(path)

	if filepath.Separator != '/' {
		path = strings.Replace(path, string(filepath.Separator), "/", -1)
	}

	if isdir && base == ".git" {
		return true
	}

	for _, pat := range i.globres {
		if pat.MatchString(path) {
			return true
		}
	}

	for _, pat := range i.res {
		if pat.MatchString(path) {
			return true
		}
	}

	return false
}

func (i *GitMatcher) Append(pats []string) {
	for _, pat := range pats {
		re, err := regexp.Compile(pat)
		if err != nil {
			errhandle(fmt.Errorf("can't compile pattern %s\n", pat), false)
			continue
		}
		i.res = append(i.res, re)
	}
}

func (i *GitMatcher) String() string {
	desc := fmt.Sprintf("Ignoring patterns from %s:", i.fp)
	if len(i.globs) > 0 {
		desc += "\n\tglobs: "
		for _, x := range i.globs {
			if strings.HasPrefix(x, i.basepath) {
				desc += x[len(i.basepath):] + " "
			} else {
				desc += x + " "
			}
		}
	}

	if len(i.res) > 0 {
		desc += "\n\tregular expressions: "
		for _, x := range i.res {
			desc += x.String() + " "
		}
	}

	return desc
}
