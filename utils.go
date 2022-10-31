package main

import "github.com/allen-b1/sus-tux/nui"

func ternaryByte(cond bool, iftrue byte, other byte) byte {
	if cond {
		return iftrue
	} else {
		return other
	}
}

func ternaryString(cond bool, t string, f string) string {
	if cond {
		return t
	} else {
		return f
	}
}

func ternaryColor(cond bool, t nui.Color, f nui.Color) nui.Color {
	if cond {
		return t
	} else {
		return f
	}
}
