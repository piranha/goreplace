package highlight

import (
	"fmt"
	"os"
	"strings"
	"regexp"
)

var Colors = map[string] int {
	"clear"				: 0,
	"reset"				: 0,
	"bold"				: 1,
    "dark"				: 2,
    "faint"				: 2,
    "underline"			: 4,
    "underscore"		: 4,
    "blink"				: 5,
    "reverse"			: 7,
    "concealed"			: 8,

    "black"				: 30,   "on_black"          : 40,
    "red"				: 31,   "on_red"            : 41,
    "green"				: 32,   "on_green"          : 42,
    "yellow"			: 33,   "on_yellow"         : 43,
    "blue"				: 34,   "on_blue"           : 44,
    "magenta"			: 35,   "on_magenta"        : 45,
    "cyan"				: 36,   "on_cyan"           : 46,
    "white"				: 37,   "on_white"          : 47,

    "bright_black"		: 90,   "on_bright_black"   : 100,
    "bright_red"		: 91,   "on_bright_red"     : 101,
    "bright_green"		: 92,   "on_bright_green"   : 102,
    "bright_yellow"		: 93,   "on_bright_yellow"  : 103,
    "bright_blue"		: 94,   "on_bright_blue"    : 104,
    "bright_magenta"	: 95,   "on_bright_magenta" : 105,
    "bright_cyan"		: 96,   "on_bright_cyan"    : 106,
    "bright_white"		: 97,   "on_bright_white"   : 107,
}

func getcolor(color string) string {
	if c, ok := Colors[color]; ok {
		return fmt.Sprintf("\033[%dm", c)
	}
	panic("Uknown color")
}

func GetColors(colors string) (cs string) {
	for _, color := range strings.Split(colors, " ", -1) {
		cs += getcolor(color)
	}
	return cs
}

func Printf(colors string, format string, a ...interface{}) {
	Reprintf(colors, regexp.MustCompile(".*"), format, a...)
}

func Reprintf(colors string, re *regexp.Regexp, format string, a ...interface{}) {
	s := re.ReplaceAllStringFunc(fmt.Sprintf(format, a...),
		func(wrap string) string {
		return GetColors(colors) + wrap + GetColors("reset")
	})
	fmt.Fprint(os.Stdout, s)
}
