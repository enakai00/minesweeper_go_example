package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync"
	"time"

	termbox "github.com/nsf/termbox-go"
)

var mu sync.Mutex // protect to wirte on screen

type key int

const (
	keyH key = iota
	keyJ
	keyK
	keyL
	keyESC
	keySpace
	keyTab
	keyNone
)

var charMap = map[int]rune{
	0: '　',
	1: '１', 2: '２', 3: '３', 4: '４',
	5: '５', 6: '６', 7: '７', 8: '８',
	9:  '・',
	10: '＊',
	11: '？',
	12: '💣',
}

type environment struct {
	size    int
	originX int
	originY int
	field   [][]int
	mines   [][]int
	cursorX int
	cursorY int
}

func (env environment) showField(showCursor bool) {
	mu.Lock()
	defer mu.Unlock()
	for x := 0; x < env.size+2; x++ {
		termbox.SetCell(env.originX+x*2, env.originY, '＃',
			termbox.ColorDefault, termbox.ColorDefault)
		termbox.SetCell(env.originX+x*2, env.originY+env.size+1, '＃',
			termbox.ColorDefault, termbox.ColorDefault)
	}
	for y := 0; y < env.size; y++ {
		termbox.SetCell(env.originX, env.originY+y+1, '＃',
			termbox.ColorDefault, termbox.ColorDefault)
		termbox.SetCell(env.originX+env.size*2+2, env.originY+y+1, '＃',
			termbox.ColorDefault, termbox.ColorDefault)
		for x := 0; x < env.size; x++ {
			fgColor := termbox.ColorDefault
			bgColor := termbox.ColorDefault
			if showCursor && x == env.cursorX && y == env.cursorY {
				fgColor = termbox.ColorWhite
				bgColor = termbox.ColorMagenta
			}
			termbox.SetCell(env.originX+x*2+2, env.originY+y+1,
				charMap[env.field[y][x]], fgColor, bgColor)
		}
	}
	termbox.Flush()
}

func (env environment) check() bool {
	for y := 0; y < env.size; y++ {
		for x := 0; x < env.size; x++ {
			if env.field[y][x] == 9 || env.field[y][x] == 11 {
				return false
			}
			if env.mines[y][x] != 1 && env.field[y][x] == 10 {
				return false
			}
		}
	}
	return true
}

func (env *environment) change(x, y int) {
	mark := env.field[y][x]
	if mark < 9 || mark > 11 {
		return
	}
	mark++
	if mark > 11 {
		mark = 9
	}
	env.field[y][x] = mark
}

func (env *environment) open(x, y int) bool {
	if env.field[y][x] >= 0 && env.field[y][x] <= 8 {
		return true
	}
	if env.mines[y][x] == 1 {
		env.field[y][x] = 12
		return false
	}

	numMines := 0
	for dy := -1; dy < 2; dy++ {
		for dx := -1; dx < 2; dx++ {
			if dx == 0 && dy == 0 ||
				x+dx < 0 || x+dx >= env.size ||
				y+dy < 0 || y+dy >= env.size {
				continue
			}
			if env.mines[y+dy][x+dx] == 1 {
				numMines++
			}
		}
	}
	env.field[y][x] = numMines
	env.showField(false)
	if numMines == 0 {
		for dy := -1; dy < 2; dy++ {
			for dx := -1; dx < 2; dx++ {
				if dx == 0 && dy == 0 ||
					x+dx < 0 || x+dx >= env.size ||
					y+dy < 0 || y+dy >= env.size {
					continue
				}
				env.open(x+dx, y+dy)
			}
		}
	}
	return true
}

func newGame(size int, level int) environment {
	env := environment{}
	env.originX, env.originY = 0, 1
	env.cursorX, env.cursorY = 0, 0
	env.size = size
	env.field = make([][]int, size)
	env.mines = make([][]int, size)

	for y := 0; y < size; y++ {
		env.field[y] = make([]int, size)
		env.mines[y] = make([]int, size)
		for x := 0; x < size; x++ {
			env.field[y][x] = 9
			env.mines[y][x] = 0
		}
	}

	// 1 <= level <= 3
	for i := 0; i < size*size*level/20; i++ {
		x, y := rand.Intn(size), rand.Intn(size)
		env.mines[y][x] = 1
	}

	return env
}

func getKey() key {
	ev := termbox.PollEvent()
	if ev.Type == termbox.EventKey {
		switch ev.Key {
		case termbox.KeyEsc, termbox.KeyCtrlC:
			return keyESC
		case termbox.KeySpace:
			return keySpace
		case termbox.KeyTab:
			return keyTab
		default:
			switch ev.Ch {
			case 'h':
				return keyH
			case 'j':
				return keyJ
			case 'k':
				return keyK
			case 'l':
				return keyL
			}
		}
	}
	return keyNone
}

func timer(done <-chan struct{}) {
	tick := time.Tick(1 * time.Second)
	elapsed := 0
	for {
		message := []string{fmt.Sprintf("Time: %dsec", elapsed)}
		drawLines(0, 0, message)
		select {
		case <-tick:
			elapsed++
		case <-done:
			return
		}
	}
}

func drawLines(x, y int, lines []string) {
	mu.Lock()
	defer mu.Unlock()
	for c, str := range lines {
		runes := []rune(str)
		for i := 0; i < len(runes); i++ {
			termbox.SetCell(x+i, y+c, runes[i],
				termbox.ColorDefault, termbox.ColorDefault)
		}
	}
	termbox.Flush()
}

func win(env environment) {
	env.showField(false)
	message := []string{"Congraturations! (Hit any key)"}
	drawLines(0, env.size+5, message)
	getKey()
}

func lose(env environment) {
	env.showField(false)
	message := []string{"Bomb! (Hit any key)"}
	drawLines(0, env.size+5, message)
	getKey()
}

func play(size, level int) {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	env := newGame(size, level)
	done := make(chan struct{})
	go timer(done)

	message := []string{
		"Move: [h][j][k][l], Mark: [Tab]",
		"Open: [SPACE], Quit: [ESC]",
	}
	drawLines(0, env.size+3, message)

	for {
		env.showField(true)
		key := getKey()
		switch key {
		case keyESC:
			return
		case keyH: // Left
			if env.cursorX > 0 {
				env.cursorX--
			}
		case keyJ: // Down
			if env.cursorY < env.size-1 {
				env.cursorY++
			}
		case keyK: // Up
			if env.cursorY > 0 {
				env.cursorY--
			}
		case keyL: // Right
			if env.cursorX < env.size-1 {
				env.cursorX++
			}
		case keyTab:
			env.change(env.cursorX, env.cursorY)
			if env.check() {
				close(done)
				win(env)
				return
			}
		case keySpace:
			if !env.open(env.cursorX, env.cursorY) {
				close(done)
				lose(env)
				return
			}
			if env.check() {
				close(done)
				win(env)
				return
			}
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	size := flag.Int("s", 10, "Board size")
	level := flag.Int("l", 2, "Game level (1-3)")
	flag.Parse()
	play(*size, *level)
}
