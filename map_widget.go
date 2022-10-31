package main

import (
	"github.com/allen-b1/sus-tux/nui"
)

const MAP_WIDTH = 128
const MAP_HEIGHT = 32

type MapWidget struct {
	X           uint16
	Y           uint16
	PlayerColor nui.Color
	Map         *Map

	PlayerX, PlayerY *uint32
	Direction        *[2]int8

	// Required
	PlayerPositions func() [][2]uint32
}

func (m *MapWidget) Draw(buf *nui.Buffer) {
	// Position of the map
	// where the top-left corner is
	offX := int32(*m.PlayerX) - MAP_WIDTH/2
	offY := int32(*m.PlayerY) - MAP_HEIGHT/2

	for x := m.X; x < m.X+MAP_WIDTH; x++ {
		for y := m.Y; y < m.Y+MAP_HEIGHT; y++ {
			idx := buf.Index(x, y)
			mapX := int32(x-m.X) + offX
			mapY := int32(y-m.Y) + offY

			var ch byte
			if mapX >= 1 && mapX < int32(m.Map.Width)-1 &&
				mapY >= 1 && mapY < int32(m.Map.Height())-1 {
				ch = m.Map.Data[int(mapX+mapY*int32(m.Map.Width))]
			} else {
				ch = 0
			}

			buf.Chars[idx] = ch
			if ch == 0 {
				buf.Chars[idx] = ' '
				buf.Formats[idx] = nui.Format{Bg: nui.LightBlack, Fg: nui.Black}
			} else if ch == ' ' {
				buf.Formats[idx] = nui.Format{Bg: nui.LightWhite, Fg: nui.LightWhite}
			} else if ch == '+' {
				buf.Formats[idx] = nui.Format{Bg: nui.LightBlack, Fg: nui.Black}
			} else {
				buf.Formats[idx] = nui.Format{Bg: nui.Magenta, Fg: nui.LightWhite}
			}
		}
	}

	positions := m.PlayerPositions()
	for playerIdx, position := range positions {
		mapX := position[0]
		mapY := position[1]

		viewX := int32(mapX) - offX
		viewY := int32(mapY) - offY
		if viewX < 0 || viewY < 0 || viewX >= MAP_WIDTH || viewY >= MAP_HEIGHT {
			continue
		}

		idx := buf.Index(uint16(viewX)+m.X, uint16(viewY)+m.Y)
		buf.Chars[idx] = 'x'
		buf.Formats[idx] = nui.Format{Fg: nui.Color(playerIdx + 1), Bg: nui.LightWhite}
	}

	buf.CursorX = m.X + MAP_WIDTH/2
	buf.CursorY = m.Y + MAP_HEIGHT/2
	buf.CursorFormat = nui.Format{Bg: m.PlayerColor, Fg: nui.LightWhite}
}

func (m *MapWidget) Focus(focus bool) {}

func (m *MapWidget) Keypress(ch byte) {
	if ch == 'w' {
		*m.Direction = [2]int8{0, -1}
	} else if ch == 's' {
		*m.Direction = [2]int8{0, 1}
	} else if ch == 'a' {
		*m.Direction = [2]int8{-1, 0}
	} else if ch == 'd' {
		*m.Direction = [2]int8{1, 0}
	} else if ch == 'q' {
		m.Direction[0] = 0
		m.Direction[1] = 0
	}
}
