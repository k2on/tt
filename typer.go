package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell"
)

type typer struct {
	Scr     tcell.Screen
	OnStart func()

	currentWordStyle    tcell.Style
	nextWordStyle       tcell.Style
	incorrectSpaceStyle tcell.Style
	incorrectStyle      tcell.Style
	correctStyle        tcell.Style
	backgroundStyle     tcell.Style
}

func NewTyper(scr tcell.Screen, fgcol, bgcol, hicol, hicol2, hicol3, errcol tcell.Color) *typer {
	def := tcell.StyleDefault.
		Foreground(fgcol).
		Background(bgcol)

	return &typer{
		Scr:                 scr,
		backgroundStyle:     def,
		correctStyle:        def.Foreground(hicol),
		currentWordStyle:    def.Foreground(hicol2),
		nextWordStyle:       def.Foreground(hicol3),
		incorrectStyle:      def.Foreground(errcol),
		incorrectSpaceStyle: def.Background(errcol),
	}
}

func (t *typer) highlight(text []cell, idx int, currentWordStyle, nextWorkStyle tcell.Style) {
	for ; idx < len(text) && text[idx].c != ' '; idx++ {
		text[idx].style = currentWordStyle
	}

	for ; idx < len(text) && text[idx].c == ' '; idx++ {
	}

	for ; idx < len(text) && text[idx].c != ' '; idx++ {
		text[idx].style = nextWorkStyle
	}
}

func (t *typer) Start(text []string) (nerrs, ncorrect int, tim time.Duration, complete bool) {
	var startTime time.Time

	for i, p := range text {
		var fn func() = nil
		if i == 0 {
			fn = func() {
				startTime = time.Now()
			}
		}

		e, c, completed := t.start(p, fn)

		nerrs += e
		ncorrect += c

		if !completed {
			tim = time.Now().Sub(startTime)
			complete = false
			return
		}
	}

	tim = time.Now().Sub(startTime)
	complete = true

	return
}

func (t *typer) start(s string, onStart func()) (int, int, bool) {
	started := false
	text := stringToCells(s)
	for i, _ := range text {
		text[i].style = t.backgroundStyle
	}

	fmt.Printf("\033[5 q")

	//Assumes original cursor shape was a block (the one true cursor shape), there doesn't appear to be a
	//good way to save/restore the shape if the user has changed it from the otcs.
	defer fmt.Printf("\033[2 q")

	t.Scr.SetStyle(t.backgroundStyle)
	idx := 0

	redraw := func() {
		//Potentially inefficient, but seems to be good enough
		drawCellsAtCenter(t.Scr, text, idx)
		t.Scr.Show()
	}

	for {
		t.highlight(text, idx, t.currentWordStyle, t.nextWordStyle)
		redraw()

		ev := t.Scr.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			t.Scr.Sync()
			t.Scr.Clear()
		case *tcell.EventKey:
			if !started {
				if onStart != nil {
					onStart()
				}

				started = true
			}

			switch ev.Key() {
			case tcell.KeyCtrlC:
				fmt.Printf("\033[2 q")
				t.Scr.Fini()
				os.Exit(1)
			case tcell.KeyEscape:
				return -1, -1, false
			case tcell.KeyCtrlL:
				t.Scr.Sync()
			case tcell.KeyCtrlH: //Control+backspace
				t.highlight(text, idx, t.backgroundStyle, t.backgroundStyle)

				if idx == 0 {
					break
				}

				idx--

				for idx > 0 && (text[idx].c == ' ' || text[idx].c == '\n') {
					text[idx].style = t.backgroundStyle
					idx--
				}

				for idx > 0 && text[idx].c != ' ' && text[idx].c != '\n' {
					text[idx].style = t.backgroundStyle
					idx--
				}

				if text[idx].c == ' ' || text[idx].c == '\n' {
					idx++
				}

				t.highlight(text, idx, t.currentWordStyle, t.nextWordStyle)
			case tcell.KeyBackspace2:
				t.highlight(text, idx, t.backgroundStyle, t.backgroundStyle)

				if idx == 0 {
					break
				}

				if idx < len(text) {
					text[idx].style = t.backgroundStyle
				}

				idx--

				for idx > 0 && text[idx].c == '\n' {
					text[idx].style = t.backgroundStyle
					idx--
				}

				t.highlight(text, idx, t.currentWordStyle, t.nextWordStyle)
			case tcell.KeyRune:
				if idx < len(text) {
					switch {
					case ev.Rune() == text[idx].c:
						text[idx].style = t.correctStyle
						idx++
					case ev.Rune() == ' ':
						for idx < len(text) && text[idx].c != ' ' && text[idx].c != '\n' {
							text[idx].style = t.incorrectStyle
							idx++
						}

						if idx < len(text) {
							text[idx].style = t.incorrectSpaceStyle
							idx++
						}
					case text[idx].c == ' ':
						text[idx].style = t.incorrectSpaceStyle
						idx++
					default:
						text[idx].style = t.incorrectStyle
						idx++
					}

					for idx < len(text) && text[idx].c == '\n' {
						idx++
					}

					t.highlight(text, idx, t.currentWordStyle, t.nextWordStyle)
				}

				if idx == len(text) {
					nerrs := 0

					for _, c := range text {
						if c.style == t.incorrectStyle || c.style == t.incorrectSpaceStyle {
							nerrs++
						}
					}

					ncorrect := len(text) - nerrs

					t.Scr.Clear()
					return nerrs, ncorrect, true
				}
			}
		}
	}
}