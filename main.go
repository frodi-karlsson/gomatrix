package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	targetFPS               = 24
	dropletGenerationChance = 0.05
	dropletMinLength        = 5
	dropletMaxLength        = 12
	dropletMaxVelocity      = 3
	dropletMinVelocity      = 1
)

var renderBuffer strings.Builder

func main() {
	// Parse flags FIRST before touching terminal state
	colorRange := parse_color_range()

	writer := bufio.NewWriter(os.Stdout)
	hide_cursor(writer)

	// 1. Save terminal state and switch to Raw Mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		panic(err)
	}

	defer func() {
		term.Restore(fd, oldState)
		writer.WriteString("\033[2J\033[H")
		show_cursor(writer)
		writer.Flush()
	}()

	w, h, err := get_term_dims()
	if err != nil {
		panic(err)
	}

	exitChan := make(chan bool)
	go func() {
		b := make([]byte, 1)
		os.Stdin.Read(b)
		exitChan <- true
	}()

	ticker := time.NewTicker(time.Second / time.Duration(targetFPS))
	defer ticker.Stop()

	var window [][]Cell
	var droplets []Droplet

	for {
		select {
		case <-exitChan:
			return

		case <-ticker.C:
			window, droplets = move_window(droplets, w, h, colorRange)
			print_window(window, writer)
			writer.Flush()
		}
	}
}

func show_cursor(writer *bufio.Writer) {
	writer.WriteString("\033[?25h")
}

func hide_cursor(writer *bufio.Writer) {
	writer.WriteString("\033[?25l")
}

func parse_color_range() ColorRange {
	color_name_to_hsl := map[string]ColorRange{
		"green": {
			hueStart:        110.0,
			hueEnd:          140.0,
			saturationStart: 0.5,
			saturationEnd:   1.0,
			lightnessStart:  0.2,
			lightnessEnd:    0.4,
		},
		"blue": {
			hueStart:        200.0,
			hueEnd:          240.0,
			saturationStart: 0.5,
			saturationEnd:   1.0,
			lightnessStart:  0.3,
			lightnessEnd:    0.5,
		},
		"red": {
			hueStart:        0.0,
			hueEnd:          10.0,
			saturationStart: 0.5,
			saturationEnd:   1.0,
			lightnessStart:  0.3,
			lightnessEnd:    0.5,
		},
	}

	color := flag.String("color", "green", "Color scheme to use (green, blue, red)")
	flag.Parse()

	return color_name_to_hsl[*color]
}

func move_window(droplets []Droplet, w, h int, colorRange ColorRange) ([][]Cell, []Droplet) {
	window := make([][]Cell, h)

	for i := range window {
		window[i] = make([]Cell, w)
		for j := range window[i] {
			window[i][j] = Cell{symbol: ' ', color: HSL{hue: 0, saturation: 0, lightness: 0}}
		}
	}

	droplets = generate_droplets(w, droplets, colorRange)
	droplets = update_droplets(droplets, window)

	return window, droplets
}

func generate_droplets(w int, droplets []Droplet, colorRange ColorRange) []Droplet {
	for x := range w {
		if can_create_droplet_at(x, droplets) && rand.Float64() < dropletGenerationChance {
			droplets = append(droplets, get_droplet_of_length(rand.IntN(dropletMaxLength-dropletMinLength+1)+dropletMinLength, x, colorRange))
		}
	}

	return droplets
}

func update_droplets(droplets []Droplet, window [][]Cell) []Droplet {
	var updated []Droplet
	for _, droplet := range droplets {
		if droplet.y < len(window) {
			draw_droplet(window, droplet)
			droplet.y += droplet.velocity
			updated = append(updated, droplet)
		}
	}
	return updated
}

func can_create_droplet_at(x int, droplets []Droplet) bool {
	for _, droplet := range droplets {
		if droplet.x == x && droplet.y <= 0 {
			return false
		}
	}
	return true
}

func draw_droplet(window [][]Cell, droplet Droplet) {
	for i, symbol := range droplet.symbols {
		y := droplet.y + i
		if y >= 0 && y < len(window) {
			// last one gets to be a little brighter
			if i == len(droplet.symbols)-1 {
				color_clone := droplet.color
				color_clone.lightness = math.Min(color_clone.lightness+0.4, 1.0)
				window[y][droplet.x] = Cell{symbol: symbol, color: color_clone}
			} else {
				window[y][droplet.x] = Cell{symbol: symbol, color: droplet.color}
			}
		}
	}
}

func hsl_to_ansi(hsl HSL) string {
	r, g, b := hsl_to_rgb(hsl)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func hsl_to_rgb(hsl HSL) (r, g, b int) {
	c := (1 - math.Abs(2*hsl.lightness-1)) * hsl.saturation
	x := c * (1 - math.Abs(math.Mod(hsl.hue/60, 2)-1))
	m := hsl.lightness - c/2

	var r1, g1, b1 float64
	switch {
	case hsl.hue < 60:
		r1, g1, b1 = c, x, 0
	case hsl.hue < 120:
		r1, g1, b1 = x, c, 0
	case hsl.hue < 180:
		r1, g1, b1 = 0, c, x
	case hsl.hue < 240:
		r1, g1, b1 = 0, x, c
	case hsl.hue < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}

	r = int((r1 + m) * 255)
	g = int((g1 + m) * 255)
	b = int((b1 + m) * 255)

	return
}

func get_droplet_of_length(length int, x int, colorRange ColorRange) Droplet {
	droplet := make([]rune, length)
	for i := range droplet {
		droplet[i] = get_random_symbol()
	}

	color := get_random_color(colorRange)
	velocity := rand.IntN(dropletMaxVelocity) + dropletMinVelocity

	return Droplet{symbols: droplet, y: -length, x: x, color: color, velocity: velocity}
}

func get_random_color(colorRange ColorRange) HSL {
	return HSL{
		hue:        colorRange.hueStart + rand.Float64()*(colorRange.hueEnd-colorRange.hueStart),
		saturation: colorRange.saturationStart + rand.Float64()*(colorRange.saturationEnd-colorRange.saturationStart),
		lightness:  colorRange.lightnessStart + rand.Float64()*(colorRange.lightnessEnd-colorRange.lightnessStart),
	}
}

func print_window(window [][]Cell, writer *bufio.Writer) {
	set_cursor_position(0, 0, writer)
	writer.WriteString(flatten_window(window))
}

func flatten_window(window [][]Cell) string {
	renderBuffer.Reset()
	for _, row := range window {
		for _, cell := range row {
			if cell.symbol != ' ' {
				renderBuffer.WriteString(hsl_to_ansi(cell.color))
			}
			renderBuffer.WriteRune(cell.symbol)
		}
		renderBuffer.WriteString("\r\n")
	}

	return renderBuffer.String()
}

func set_cursor_position(x, y int, writer *bufio.Writer) {
	fmt.Fprintf(writer, "\033[%d;%dH", y, x)
}

func get_term_dims() (int, int, error) {
	width, height, err := term.GetSize(0)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}

func get_random_symbol() rune {
	symbols := get_symbols()
	return symbols[rand.IntN(len(symbols))]
}

func get_symbols() []rune {
	symbols := []rune{}

	symbols = append(symbols, get_symbols_in_range('0', '9')...)
	symbols = append(symbols, '-', '_', '.', '~')
	symbols = append(symbols, '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=')

	for _, r := range []rune("ﾊﾐﾋｰｳｼﾅﾓﾆｻﾜﾂｵﾘｱﾎﾃﾏｹﾒｴｶｷﾑﾕﾗｾﾈｽﾀﾇﾍ") {
		symbols = append(symbols, r)
	}

	for _, r := range []rune("THEMATRIXZ") {
		symbols = append(symbols, r)
	}

	symbols = append(symbols, 'Z')

	return symbols
}

func get_symbols_in_range(start, end rune) []rune {
	var symbols []rune

	for r := start; r <= end; r++ {
		symbols = append(symbols, r)
	}

	return symbols
}

type Droplet struct {
	symbols  []rune
	x        int
	y        int
	color    HSL
	velocity int
}

type HSL struct {
	hue        float64
	saturation float64
	lightness  float64
}

type Cell struct {
	symbol rune
	color  HSL
}

type ColorRange struct {
	hueStart        float64
	hueEnd          float64
	saturationStart float64
	saturationEnd   float64
	lightnessStart  float64
	lightnessEnd    float64
}
