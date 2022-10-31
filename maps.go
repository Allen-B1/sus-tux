package main

import (
	"strings"
)

type Map struct {
	Data  []byte
	Width uint32
}

func (m *Map) Height() uint32 {
	return uint32(len(m.Data)) / m.Width
}

func ternary(cond bool, iftrue byte, other byte) byte {
	if cond {
		return iftrue
	} else {
		return other
	}
}

func NewMap(data string) *Map {
	m := new(Map)

	lines := strings.Split(data, "\n")
	width := len(lines[0])
	height := len(lines)

	m.Width = uint32(3 * width)
	m.Data = make([]byte, width*3*height*3)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			c := lines[y][x]

			wallLeft := x == 0 || lines[y][x-1] == c
			wallRight := x == width-1 || lines[y][x+1] == c
			wallTop := y == 0 || lines[y-1][x] == c
			wallBottom := y == height-1 || lines[y+1][x] == c

			m.Data[3*y*int(m.Width)+3*x] = ' '
			m.Data[3*y*int(m.Width)+3*x+1] = ternary(wallTop, c, ' ')
			m.Data[3*y*int(m.Width)+3*x+2] = ' '
			m.Data[(3*y+1)*int(m.Width)+3*x] = ternary(wallLeft, c, ' ')
			m.Data[(3*y+1)*int(m.Width)+3*x+1] = c
			m.Data[(3*y+1)*int(m.Width)+3*x+2] = ternary(wallRight, c, ' ')
			m.Data[(3*y+2)*int(m.Width)+3*x] = ' '
			m.Data[(3*y+2)*int(m.Width)+3*x+1] = ternary(wallBottom, c, ' ')
			m.Data[(3*y+2)*int(m.Width)+3*x+2] = ' '
		}
	}

	//	for i, c := range m.Data {
	//		if i%int(m.Width) == 0 {
	//			fmt.Println()
	//		}
	//		fmt.Printf("%c", c)
	//	}

	return m
}
