package terminal

import (
	"image"
	"image/color"
	"time"

	"github.com/nsf/termbox-go"
	"github.com/zaggash/go-rpi-rgb-led-matrix/terminal/pxl"
)

type Terminal struct {
	Img    *image.RGBA
	Width  int
	Height int
}

func NewTerminal(w, h int, autoInit bool) *Terminal {
	t := &Terminal{
		Width:  w,
		Height: h,
		Img:    image.NewRGBA(image.Rect(0, 0, w, h)),
	}

	if autoInit {
		t.Init()
	}

	return t
}

// Init initialize the emulator, creating a new Window and waiting until is
// painted. If something goes wrong the function panics
func (t *Terminal) Init() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetOutputMode(termbox.Output256)

	go func() {
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				if ev.Key == termbox.KeyEsc || ev.Ch == 'q' {
					t.Close()
				}
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
}

func (t *Terminal) Geometry() (width, height int) {
	return t.Width, t.Height
}

func (t *Terminal) position(x, y int) int {
	return x + (y * t.Width)
}

func (t *Terminal) Apply(leds []color.Color) error {
	for position, l := range leds {
		t.Set(position, l)
	}

	return t.Render()
}

func (t *Terminal) Render() error {
	pxl.DisplayImage(t.Img)
	return nil
}

func (t *Terminal) At(position int) color.Color {
	posY := position / t.Width
	posX := position % t.Width

	return t.Img.At(posX, posY)
}

func (t *Terminal) Set(position int, c color.Color) {
	posY := position / t.Width
	posX := position % t.Width

	t.Img.Set(posX, posY, color.RGBAModel.Convert(c))
}

func (t *Terminal) Close() error {
	termbox.Close()
	return nil
}

// Those new functions have no use with the emulator
func (t *Terminal) GetBrightness() int {
	return 0
}

func (t *Terminal) SetBrightness(brightness int) {}
