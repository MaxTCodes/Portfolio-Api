package visuals

import (
	"fmt"
	"github.com/mattn/go-runewidth"
	"strconv"
)

func StartUpMsg(version, adminPath string) {

	center := func(s string, width int) string {
		const padDiv = 2
		pad := strconv.Itoa((width - len(s)) / padDiv)
		str := fmt.Sprintf("%"+pad+"s", " ")
		str += s
		str += fmt.Sprintf("%"+pad+"s", " ")
		if len(str) < width {
			str += " "
		}
		return str
	}

	centerValue := func(s string, width int) string {
		const padDiv = 2
		pad := strconv.Itoa((width - runewidth.StringWidth(s)) / padDiv)
		str := fmt.Sprintf("%"+pad+"s", " ")
		str += s
		str += fmt.Sprintf("%"+pad+"s", " ")
		if runewidth.StringWidth(s)-10 < width && runewidth.StringWidth(s)%2 == 0 {
			// add an ending space if the length of str is even and str is not too long
			str += " "
		}
		return str
	}

	const lineLen = 60
	mainLogo := " ┌───────────────────────────────────────────────────────────────┐\n"
	mainLogo += " │ " + centerValue(fmt.Sprintf("Max's Portfolio Backend - %s", version), lineLen) + " │\n"
	mainLogo += " │ " + center("Powered by Go Fiber", lineLen+1) + " │\n"
	mainLogo += " │                                                               │\n"
	mainLogo += " │                                                               │\n"
	mainLogo += " │ " + centerValue(fmt.Sprintf("Login Path: https://api.maxthakur.xyz/%s", adminPath), lineLen+2) + " │\n"
	mainLogo += " │                                                               │\n"
	mainLogo += " └───────────────────────────────────────────────────────────────┘\n"

	fmt.Println(mainLogo)
}
