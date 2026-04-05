package core

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

const (
	VERSION = "4.0.0"
)

func putAsciiArt(s string) {
	for _, c := range s {
		d := string(c)
		switch {
		case d == "\n":
			color.Unset()
			fmt.Println()
		case d == " ":
			color.Unset()
			fmt.Print(d)
		case d == "╔" || d == "╗" || d == "╚" || d == "╝" || d == "╠" || d == "╣" || d == "╦" || d == "╩" || d == "╬":
			color.Set(color.FgHiCyan)
			fmt.Print(d)
		case d == "═":
			color.Set(color.FgCyan)
			fmt.Print(d)
		case d == "║":
			color.Set(color.FgHiBlue)
			fmt.Print(d)
		case d == "█":
			color.Set(color.FgHiRed)
			fmt.Print(d)
		case d == "▓":
			color.Set(color.FgRed)
			fmt.Print(d)
		case d == "▒":
			color.Set(color.FgHiYellow)
			fmt.Print(d)
		case d == "░":
			color.Set(color.FgYellow)
			fmt.Print(d)
		case d == "▄" || d == "▀":
			color.Set(color.FgHiBlack)
			fmt.Print(d)
		case d == "☠":
			color.Set(color.FgHiYellow)
			fmt.Print(d)
		case d == "⚡":
			color.Set(color.FgHiYellow)
			fmt.Print(d)
		case d == "◆" || d == "◈":
			color.Set(color.FgHiMagenta)
			fmt.Print(d)
		case strings.Contains("𝗣𝗥𝗢𝗩𝗘𝗥𝗦𝗜𝗢𝗡", d):
			color.Set(color.FgHiYellow)
			fmt.Print(d)
		default:
			color.Set(color.FgHiWhite)
			fmt.Print(d)
		}
	}
	color.Unset()
}

func printUpdateName() {
	nameClr := color.New(color.FgYellow)
	txt := nameClr.Sprintf("")
	fmt.Fprintf(color.Output, "%s", txt)
}

func printOneliner1() {
	handleClr := color.New(color.FgHiCyan)
	versionClr := color.New(color.FgHiGreen)
	textClr := color.New(color.FgHiWhite)
	dimClr := color.New(color.FgHiBlack)
	accentClr := color.New(color.FgHiMagenta)
	spc := strings.Repeat(" ", 10-len(VERSION))
	txt := dimClr.Sprintf("   ◈ ") +
		textClr.Sprintf("x-tymus") +
		dimClr.Sprintf(" ⟨") +
		handleClr.Sprintf("@x-tymus") +
		dimClr.Sprintf("⟩") +
		spc +
		accentClr.Sprintf("◆ ") +
		dimClr.Sprintf("version ") +
		versionClr.Sprintf("%s", VERSION)
	fmt.Fprintf(color.Output, "%s", txt)
}

func printOneliner2() {
	textClr := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)
	white := color.New(color.FgWhite)
	txt := textClr.Sprintf("") + red.Sprintf("") + white.Sprintf("") + textClr.Sprintf("") + red.Sprintf("")
	fmt.Fprintf(color.Output, "%s", txt)
}

func Banner() {
	fmt.Println()

	putAsciiArt(`
╔═══════════════════════════════════════════════════════════════════════════╗
║  ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄  ║
║                                                                           ║
║   ██╗  ██╗ ◆  ████████╗██╗   ██╗███╗   ███╗██╗   ██╗███████╗            ║
║   ╚██╗██╔╝ ◆  ╚══██╔══╝╚██╗ ██╔╝████╗ ████║██║   ██║██╔════╝            ║
║    ╚███╔╝  ◆     ██║    ╚████╔╝ ██╔████╔██║██║   ██║███████╗             ║
║    ██╔██╗  ◆     ██║     ╚██╔╝  ██║╚██╔╝██║██║   ██║╚════██║             ║
║   ██╔╝ ██╗ ◆     ██║      ██║   ██║ ╚═╝ ██║╚██████╔╝███████║             ║
║   ╚═╝  ╚═╝ ◆     ╚═╝      ╚═╝   ╚═╝     ╚═╝ ╚═════╝ ╚══════╝            ║
║                                                                           ║
║  ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀  ║
║      ☠  ░▒▓█  𝗣𝗥𝗢  𝗩𝗘𝗥𝗦𝗜𝗢𝗡  █▓▒░  ☠                  ⚡ v4.0.0 ⚡   ║
╚═══════════════════════════════════════════════════════════════════════════╝
`)

	printUpdateName()
	fmt.Println()
	printOneliner1()
	fmt.Println()
	printOneliner2()
	fmt.Println()
	fmt.Println()
}
