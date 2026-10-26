package graph

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-sixel"
)

// SixelOptions controls the raster size and title of a Sixel chart.
type SixelOptions struct {
	Width  int
	Height int
	Title  string
}

const (
	sixelCellWidthPx   = 10
	sixelMinWidthPx    = 320
	sixelMaxWidthPx    = 2000
	sixelDefaultHeight = 360
	sixelMarginPx      = 12
	sixelLineThickness = 2
)

var (
	sixelBackground = color.RGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff} //nolint:gochecknoglobals // Immutable palette; color.RGBA cannot be a const.
	sixelGridColor  = color.RGBA{R: 0x3a, G: 0x3a, B: 0x3a, A: 0xff} //nolint:gochecknoglobals // Immutable palette; color.RGBA cannot be a const.
)

// DefaultSixelOptions sizes the image from the terminal width: about 10px per column.
func DefaultSixelOptions() SixelOptions {
	cols, _ := getTerminalSize()
	width := min(max(cols*sixelCellWidthPx, sixelMinWidthPx), sixelMaxWidthPx)
	return SixelOptions{Width: width, Height: sixelDefaultHeight}
}

// RenderSixel draws the chart as a Sixel image, followed by a text legend.
// Instant queries render as horizontal bars; range queries render as lines.
func RenderSixel(w io.Writer, data *ChartData, opts SixelOptions) error {
	if data == nil || len(data.Series) == 0 {
		_, err := fmt.Fprintln(w, "No data to display")
		return err
	}

	title := opts.Title
	if title == "" {
		title = data.Title
	}
	if title != "" {
		fmt.Fprintln(w, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D7FF")).Render(title))
	}

	width, height := opts.Width, opts.Height
	if width <= 0 {
		width = sixelMinWidthPx
	}
	if height <= 0 {
		height = sixelDefaultHeight
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: sixelBackground}, image.Point{}, draw.Src)

	instant := data.IsInstantQuery()
	if instant {
		drawSixelBars(img, data)
	} else {
		drawSixelLines(img, data)
	}

	if err := sixel.NewEncoder(w).Encode(img); err != nil {
		return fmt.Errorf("encoding sixel image: %w", err)
	}
	fmt.Fprintln(w)

	if instant {
		if legend := renderBarLegend(data.Series); legend != "" {
			fmt.Fprintln(w, legend)
		}
		return nil
	}

	if legend := renderLegend(data.Series); legend != "" {
		fmt.Fprintln(w, legend)
	}
	if caption := sixelRangeCaption(data); caption != "" {
		fmt.Fprintln(w, caption)
	}
	return nil
}

func drawSixelLines(img *image.RGBA, data *ChartData) {
	minT, maxT, minY, maxY, ok := finiteBounds(data)
	if !ok {
		return
	}
	if minT.Equal(maxT) {
		minT = minT.Add(-time.Minute)
		maxT = maxT.Add(time.Minute)
	}
	yRange := maxY - minY
	if yRange == 0 {
		yRange = 1
	}
	padding := yRange * 0.1
	minY -= padding
	maxY += padding

	plot := sixelPlotRect(img)
	drawSixelGrid(img, plot)

	tSpan := float64(maxT.Sub(minT))
	ySpan := maxY - minY
	for i, s := range data.Series {
		c := toRGBA(seriesColor(s, i))
		var prevX, prevY int
		havePrev := false
		for _, p := range s.Points {
			if !isFiniteValue(p.Value) {
				havePrev = false
				continue
			}
			x := plot.Min.X + int(math.Round(float64(p.Time.Sub(minT))/tSpan*float64(plot.Dx()-1)))
			y := plot.Max.Y - 1 - int(math.Round((p.Value-minY)/ySpan*float64(plot.Dy()-1)))
			if havePrev {
				drawSixelSegment(img, prevX, prevY, x, y, c)
			} else {
				stampSixelDot(img, x, y, c)
			}
			prevX, prevY, havePrev = x, y, true
		}
	}
}

func drawSixelBars(img *image.RGBA, data *ChartData) {
	type bar struct {
		color color.RGBA
		value float64
	}
	var bars []bar
	maxValue := 0.0
	for i, s := range data.Series {
		if len(s.Points) == 0 {
			continue
		}
		v := s.Points[0].Value
		if !isFiniteValue(v) {
			v = 0
		}
		bars = append(bars, bar{color: toRGBA(seriesColor(s, i)), value: v})
		maxValue = math.Max(maxValue, v)
	}
	if len(bars) == 0 {
		return
	}
	if maxValue <= 0 {
		maxValue = 1
	}

	plot := sixelPlotRect(img)
	rowHeight := max(plot.Dy()/len(bars), 1)
	barHeight := max(rowHeight*2/3, 1)
	for idx, b := range bars {
		top := plot.Min.Y + idx*rowHeight + (rowHeight-barHeight)/2
		length := int(math.Round(math.Max(b.value, 0) / maxValue * float64(plot.Dx())))
		fillSixelRect(img, image.Rect(plot.Min.X, top, plot.Min.X+length, top+barHeight), b.color)
	}
}

func drawSixelGrid(img *image.RGBA, plot image.Rectangle) {
	const lines = 4
	for i := 1; i <= lines; i++ {
		y := plot.Min.Y + i*plot.Dy()/(lines+1)
		fillSixelRect(img, image.Rect(plot.Min.X, y, plot.Max.X, y+1), sixelGridColor)
	}
}

func sixelPlotRect(img *image.RGBA) image.Rectangle {
	b := img.Bounds()
	return image.Rect(sixelMarginPx, sixelMarginPx, b.Dx()-sixelMarginPx, b.Dy()-sixelMarginPx)
}

func fillSixelRect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(img, r.Intersect(img.Bounds()), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

// drawSixelSegment draws a line with Bresenham's algorithm, stamping a square brush at each step.
func drawSixelSegment(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		stampSixelDot(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func stampSixelDot(img *image.RGBA, x, y int, c color.RGBA) {
	fillSixelRect(img, image.Rect(x, y, x+sixelLineThickness, y+sixelLineThickness), c)
}

// finiteBounds returns the raw time and value extent, ignoring NaN and infinite values.
func finiteBounds(data *ChartData) (time.Time, time.Time, float64, float64, bool) {
	var minT, maxT time.Time
	var minY, maxY float64
	found := false
	for _, s := range data.Series {
		for _, p := range s.Points {
			if !isFiniteValue(p.Value) {
				continue
			}
			if !found {
				minT, maxT = p.Time, p.Time
				minY, maxY = p.Value, p.Value
				found = true
				continue
			}
			minT = earlierTime(minT, p.Time)
			maxT = laterTime(maxT, p.Time)
			minY = math.Min(minY, p.Value)
			maxY = math.Max(maxY, p.Value)
		}
	}
	return minT, maxT, minY, maxY, found
}

func sixelRangeCaption(data *ChartData) string {
	minT, maxT, minY, maxY, ok := finiteBounds(data)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s to %s  (y: %.4g to %.4g)",
		minT.Format("2006-01-02 15:04:05"), maxT.Format("2006-01-02 15:04:05"), minY, maxY)
}

func isFiniteValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func toRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func earlierTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
