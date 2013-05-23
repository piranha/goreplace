// (c) 2011-2013 Alexander Solovyov
// under terms of ISC license

package fnmatch

import (
	"bytes"
	"regexp"
	"strings"
)

var matchCache = map[string]*regexp.Regexp{}

func Match(pattern string, path string) (bool, error) {
	re, ok := matchCache[pattern]
	if !ok {
		translated := translate(pattern)

		// declare err separately so that re is not overriden with :=
		var err error
		re, err = regexp.Compile(translated)
		if err != nil {
			return false, err
		}
		matchCache[pattern] = re
	}

	return re.Match([]byte(path)), nil
}

func translate(pat string) string {
	var b bytes.Buffer
	i := 0
	n := len(pat)

	for i < n {
		c := pat[i]
		i += 1

		switch c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			j := i
			if j < n && (pat[j] == '!' || pat[j] == ']') {
				j += 1
			}
			for j < n && pat[j] != ']' {
				j += 1
			}

			if j >= n {
				b.WriteString("\\[")
			} else {
				stuff := strings.Replace(pat[i:j], "\\", "\\\\", -1)
				i = j + 1
				if stuff[0] == '!' {
					stuff = "^" + stuff[1:]
				} else if stuff[0] == '^' {
					stuff = "\\" + stuff
				}
				b.WriteString("[")
				b.WriteString(stuff)
				b.WriteString("]")
			}
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}

	return b.String()
}
