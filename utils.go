// (c) 2011-2014 Alexander Solovyov
// under terms of ISC License

package main

import (
	"fmt"
	"github.com/wsxiaoys/terminal/color"
)

func ColorPrintf(colorfmt, plainfmt string, args... interface{}) {
	if NoColors {
		fmt.Printf(plainfmt, args...)
	} else {
		color.Printf(colorfmt, args...)
	}
}

func ColorSprintf(colorfmt, plainfmt string, args... interface{}) string {
	if NoColors {
		return fmt.Sprintf(plainfmt, args...)
	} else {
		return color.Sprintf(colorfmt, args...)
	}
}
