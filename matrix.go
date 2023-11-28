package rgbmatrix

/*
#cgo CFLAGS: -std=c99 -I${SRCDIR}/lib/rpi-rgb-led-matrix/include -DSHOW_REFRESH_RATE
#cgo LDFLAGS: -lrgbmatrix -L${SRCDIR}/lib/rpi-rgb-led-matrix/lib -lstdc++ -lm
#include <led-matrix-c.h>

void led_matrix_swap(struct RGBLedMatrix *matrix, struct LedCanvas *offscreen_canvas,
                     int width, int height, const uint32_t pixels[]) {


  int i, x, y;
  uint32_t color;
  for (x = 0; x < width; ++x) {
    for (y = 0; y < height; ++y) {
      i = x + (y * width);
      color = pixels[i];

      led_canvas_set_pixel(offscreen_canvas, x, y,
        (color >> 16) & 255, (color >> 8) & 255, color & 255);
    }
  }

  offscreen_canvas = led_matrix_swap_on_vsync(matrix, offscreen_canvas);
}

void set_show_refresh_rate(struct RGBLedMatrixOptions *o, int show_refresh_rate) {
  o->show_refresh_rate = show_refresh_rate != 0 ? 1 : 0;
}

void set_disable_hardware_pulsing(struct RGBLedMatrixOptions *o, int disable_hardware_pulsing) {
  o->disable_hardware_pulsing = disable_hardware_pulsing != 0 ? 1 : 0;
}

void set_inverse_colors(struct RGBLedMatrixOptions *o, int inverse_colors) {
  o->inverse_colors = inverse_colors != 0 ? 1 : 0;
}
*/
import "C"
import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"unsafe"

	"github.com/zaggash/go-rpi-rgb-led-matrix/emulator"
	"github.com/zaggash/go-rpi-rgb-led-matrix/terminal"
)

var DefaultRtConfig = RuntimeConfig{
	GPIOSlowdown: 0,
}

// DefaultConfig default configuration
var DefaultConfig = HardwareConfig{
	GPIOMapping:            "regular",
	Rows:                   32,
	Cols:                   32,
	ChainLength:            1,
	Parallel:               1,
	PanelType:              "",
	Multiplexing:           0,
	RowAddressType:         0,
	PixelMapperConfig:      "",
	Brightness:             100,
	PWMBits:                11,
	ShowRefreshRate:        false,
	LimitRefresh:           0,
	ScanMode:               Progressive,
	PWMLSBNanoseconds:      130,
	PWMDitherBits:          0,
	DisableHardwarePulsing: false,
	InverseColors:          false,
	RGBSequence:            "RGB",
}

// RuntimeConfig
type RuntimeConfig struct {
	// The Raspberry Pi starting with Pi2 are putting out data too fast.
	// In this case, you want to slow down writing to GPIO.
	// Zero for this parameter means 'no slowdown'.
	// The default 1 (one) typically works fine
	// You have to even go further by setting it to 2 (two).
	// If you have a Raspberry Pi with a slower processor (Model A, A+, B+, Zero), then a value of 0 (zero) might work and is desirable.
	//A Raspberry Pi 3 or Pi4 might even need higher values for the panels to be happy.
	GPIOSlowdown int
}

// HardwareConfig rgb-led-matrix configuration
type HardwareConfig struct {
	// This can have values such as:
	// * regular          -> The standard mapping of this library
	// * adafruit-hat     -> The Adafruit HAT/Bonnet, that uses this library
	// * adafruit-hat-pwm -> Adafruit HAT with the anti-flicker hardware mod
	// * compute-module   -> Additional 3 parallel chains can be used with the Compute Module.
	// https://github.com/hzeller/rpi-rgb-led-matrix/blob/master/wiring.md#alternative-hardware-mappings
	GPIOMapping string

	// Rows the number of rows supported by the display, so 32 or 16.
	Rows int

	// Cols the number of columns supported by the display, so 32 or 64 .
	Cols int

	// Number of daisy-chained panels.
	ChainLength int

	// Parallel is the number of parallel chains connected to the Pi; in old Pis
	// with 26 GPIO pins, that is 1, in newer Pis with 40 interfaces pins, that
	// can also be 2 or 3. The effective number of pixels in vertical direction is
	// then thus rows * parallel.
	Parallel int

	// Some panels use a different chip-set that requires some initialization.
	// If you don't see any output on your panel, try to set FM6126A
	// Some panels have the FM6127 chip, which is also an option.
	PanelType string

	// The outdoor panels have different multiplexing which allows them to be faster and brighter,
	// but by default their output looks jumbled up.
	// They require some pixel-mapping of which there are a few types you can try and hopefully
	// one of them works for your panel; The default=0 is no mapping ('standard' panels),
	//  while 1, 2, ... are different mappings to try
	// Mux type: 0=direct; 1=Stripe; 2=Checkered...
	Multiplexing int

	// This option is useful for certain 64x64 or 32x16 panels.
	// For 64x64 panels, that only have an A and B address line, you'd use RowAddressType to 1.
	// This is only tested with one panel so far, so if it doesn't work for you, please send a pull request.
	// For 32x16 outdoor panels, that have have 4 address line (A, B, C, D), it is necessary to use RowAddressType to 2.
	RowAddressType int

	// A string describing a sequence of pixel mappers that should be applied
	// to this matrix.
	// Semicolon-separated list of pixel-mappers to arrange pixels.
	// Mapping the logical layout of your boards to your physical arrangement.
	// https://github.com/hzeller/rpi-rgb-led-matrix/blob/master/examples-api-use#remapping-coordinates
	PixelMapperConfig string

	// Brightness is the initial brightness of the panel in percent. Valid range
	// is 1..100
	Brightness int

	// Set PWM bits used for output. Default is 11, but if you only deal with
	// limited comic-colors, 1 might be sufficient. Lower require less CPU and
	// increases refresh-rate.
	PWMBits int

	// This shows the current refresh rate of the LED panel
	// the time to refresh a full picture.
	ShowRefreshRate bool

	// This allows to limit the refresh rate to a particular frequency to approach a fixed refresh rate.
	// The refresh rate will now be adapted to always reach this value between frames,
	// so faster refreshes will be slowed down, but the occasional delayed frame will fit into the time-window as well,
	// thus reducing visible brightness fluctuations.
	// You can play with value a little and reduce until you find a good balance between refresh rate and flicker suppression.
	LimitRefresh int

	// This switches from progressive scan and interlaced scan.
	// The latter might look be a little nicer when you have a very low refresh rate
	// but typically it is more annoying because of the comb-effect
	// 0 = progressive; 1 = interlaced (Default: 0).
	ScanMode ScanMode

	// Change the base time-unit for the on-time in the lowest significant bit in
	// nanoseconds.  Higher numbers provide better quality (more accurate color,
	// less ghosting), but have a negative impact on the frame rate.
	// Good values for full-color display (PWM=11) are somewhere between 100 and 300.
	// If you use reduced bit color (e.g. PWM=1) and have sharp contrast applications,
	// then higher values might be good to minimize ghosting.
	PWMLSBNanoseconds int

	// The lower bits can be time dithered
	// i.e. their brightness contribution is achieved by only showing them some frames
	// This will allow higher refresh rate (or same refresh rate with increased PWMLSBNanoseconds).
	PWMDitherBits int

	// Disable the PWM hardware subsystem to create pulses. Typically, you don't
	// want to disable hardware pulsing, this is mostly for debugging and figuring
	// out if there is interference with the sound system.
	// This won't do anything if output enable is not connected to GPIO 18 in
	// non-standard wirings.
	DisableHardwarePulsing bool

	// Switch if your matrix has inverse colors on.
	InverseColors bool

	// These are if you have a different kind of LED panel where the Red, Green and Blue LEDs are mixed up
	// Default: "RGB"
	RGBSequence string
}

func (c *HardwareConfig) geometry() (width, height int) {
	return c.Cols * c.ChainLength, c.Rows * c.Parallel
}

func (rt *RuntimeConfig) toC() *C.struct_RGBLedRuntimeOptions {
	rto := &C.struct_RGBLedRuntimeOptions{}
	rto.gpio_slowdown = C.int(rt.GPIOSlowdown)
	return rto
}

func (c *HardwareConfig) toC() *C.struct_RGBLedMatrixOptions {
	o := &C.struct_RGBLedMatrixOptions{}
	o.hardware_mapping = C.CString(c.GPIOMapping)
	o.rows = C.int(c.Rows)
	o.cols = C.int(c.Cols)
	o.chain_length = C.int(c.ChainLength)
	o.parallel = C.int(c.Parallel)
	o.panel_type = C.CString(c.PanelType)
	o.multiplexing = C.int(c.Multiplexing)
	o.row_address_type = C.int(c.RowAddressType)
	o.pixel_mapper_config = C.CString(c.PixelMapperConfig)
	o.brightness = C.int(c.Brightness)
	o.pwm_bits = C.int(c.PWMBits)
	o.limit_refresh_rate_hz = C.int(c.LimitRefresh)
	o.scan_mode = C.int(c.ScanMode)
	o.pwm_lsb_nanoseconds = C.int(c.PWMLSBNanoseconds)
	o.pwm_dither_bits = C.int(c.PWMDitherBits)
	o.led_rgb_sequence = C.CString(c.RGBSequence)

	if c.ShowRefreshRate {
		C.set_show_refresh_rate(o, C.int(1))
	} else {
		C.set_show_refresh_rate(o, C.int(0))
	}

	if c.DisableHardwarePulsing {
		C.set_disable_hardware_pulsing(o, C.int(1))
	} else {
		C.set_disable_hardware_pulsing(o, C.int(0))
	}

	if c.InverseColors {
		C.set_inverse_colors(o, C.int(1))
	} else {
		C.set_inverse_colors(o, C.int(0))
	}

	return o
}

type ScanMode int8

const (
	Progressive ScanMode = 0
	Interlaced  ScanMode = 1
)

// RGBLedMatrix matrix representation for ws281x
type RGBLedMatrix struct {
	Config   *HardwareConfig
	RtConfig *RuntimeConfig
	height   int
	width    int
	matrix   *C.struct_RGBLedMatrix
	buffer   *C.struct_LedCanvas
	leds     []C.uint32_t
}

const MatrixEmulatorENV = "MATRIX_EMULATOR"
const TerminalMatrixEmulatorENV = "MATRIX_TERMINAL_EMULATOR"

// NewRGBLedMatrix returns a new matrix using the given size and config
func NewRGBLedMatrix(config *HardwareConfig, rtconfig *RuntimeConfig) (c Matrix, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("error creating matrix: %v", r)
			}
		}
	}()

	if isMatrixEmulator() {
		return buildMatrixEmulator(config), nil
	}
	if isTerminalMatrixEmulator() {
		return buildTerminalMatrixEmulator(config), nil
	}

	//m := C.led_matrix_create_from_options(config.toC(), nil, nil)
	m := C.led_matrix_create_from_options_and_rt_options(config.toC(), rtconfig.toC())

	b := C.led_matrix_create_offscreen_canvas(m)

	var w, h C.int
	C.led_canvas_get_size(b, &w, &h)

	c = &RGBLedMatrix{
		Config:   config,
		RtConfig: rtconfig,
		width:    int(w), height: int(h),
		matrix: m,
		buffer: b,
		leds:   make([]C.uint32_t, int(w)*int(h)),
	}
	if m == nil {
		return nil, fmt.Errorf("unable to allocate memory")
	}

	return c, nil
}

func isMatrixEmulator() bool {
	return os.Getenv(MatrixEmulatorENV) == "1"
}

func isTerminalMatrixEmulator() bool {
	return os.Getenv(TerminalMatrixEmulatorENV) == "1"
}

func buildMatrixEmulator(config *HardwareConfig) Matrix {
	w, h := config.geometry()
	return emulator.NewEmulator(w, h, emulator.DefaultPixelPitch, true)
}

func buildTerminalMatrixEmulator(config *HardwareConfig) Matrix {
	w, h := config.geometry()
	if strings.Contains(config.PixelMapperConfig, "U-mapper") {
		w /= 2
		h *= 2
	}
	return terminal.NewTerminal(w, h, true)
}

// Initialize initialize library, must be called once before other functions are
// called.
func (c *RGBLedMatrix) Initialize() error {
	return nil
}

// Geometry returns the width and the height of the matrix
func (c *RGBLedMatrix) Geometry() (width, height int) {
	return c.width, c.height
}

// Apply set all the pixels to the values contained in leds
func (c *RGBLedMatrix) Apply(leds []color.Color) error {
	for position, l := range leds {
		c.Set(position, l)
	}

	return c.Render()
}

// Render update the display with the data from the LED buffer
func (c *RGBLedMatrix) Render() error {
	w, h := c.Geometry()

	C.led_matrix_swap(
		c.matrix,
		c.buffer,
		C.int(w), C.int(h),
		(*C.uint32_t)(unsafe.Pointer(&c.leds[0])),
	)

	c.leds = make([]C.uint32_t, w*h)
	return nil
}

// At return an Color which allows access to the LED display data as
// if it were a sequence of 24-bit RGB values.
func (c *RGBLedMatrix) At(position int) color.Color {
	return uint32ToColor(c.leds[position])
}

// Set set LED at position x,y to the provided 24-bit color value.
func (c *RGBLedMatrix) Set(position int, color color.Color) {
	c.leds[position] = C.uint32_t(colorToUint32(color))
}

// Close finalizes the ws281x interface
func (c *RGBLedMatrix) Close() error {
	C.led_matrix_delete(c.matrix)
	return nil
}

// GetBrightness returns the current brightness setting of the matrix
func (c *RGBLedMatrix) GetBrightness() int {
	return int(C.led_matrix_get_brightness(c.matrix))
}

// SetBrightness sets a new brightness setting to the matrix
func (c *RGBLedMatrix) SetBrightness(brightness int) {
	C.led_matrix_set_brightness(c.matrix, C.uchar(brightness))
}

func colorToUint32(c color.Color) uint32 {
	if c == nil {
		return 0
	}

	// A color's RGBA method returns values in the range [0, 65535]
	red, green, blue, _ := c.RGBA()
	return (red>>8)<<16 | (green>>8)<<8 | blue>>8
}

func uint32ToColor(u C.uint32_t) color.Color {
	return color.RGBA{
		uint8(u>>16) & 255,
		uint8(u>>8) & 255,
		uint8(u>>0) & 255,
		0,
	}
}
