package graph_test

import (
	"bytes"
	"image"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/grafana/gcx/internal/graph"
	"github.com/mattn/go-sixel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderSixelEmitsDecodableImageOfRequestedSize(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &graph.ChartData{
		Title: "requests",
		Series: []graph.Series{{
			Name: "rate",
			Points: []graph.Point{
				{Time: t0, Value: 1},
				{Time: t0.Add(time.Minute), Value: 3},
				{Time: t0.Add(2 * time.Minute), Value: math.NaN()},
				{Time: t0.Add(3 * time.Minute), Value: 2},
			},
		}},
	}
	opts := graph.SixelOptions{Width: 320, Height: 160}

	var buf bytes.Buffer
	require.NoError(t, graph.RenderSixel(&buf, data, opts))

	out := buf.String()
	start := strings.Index(out, "\x1bP")
	end := strings.LastIndex(out, "\x1b\\")
	require.GreaterOrEqual(t, start, 0, "output must contain a Sixel DCS introducer")
	require.Greater(t, end, start, "output must end with the Sixel string terminator")

	var img image.Image
	require.NoError(t, sixel.NewDecoder(strings.NewReader(out[start:end+2])).Decode(&img))
	assert.Equal(t, opts.Width, img.Bounds().Dx())
	assert.Equal(t, opts.Height, img.Bounds().Dy())
	assert.Contains(t, out, "requests")
	assert.Contains(t, out, "rate")
}

func TestRenderSixelBarChartForInstantQuery(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &graph.ChartData{
		Series: []graph.Series{
			{Name: "a", Points: []graph.Point{{Time: t0, Value: 5}}},
			{Name: "b", Points: []graph.Point{{Time: t0, Value: 2}}},
		},
	}
	require.True(t, data.IsInstantQuery())

	var buf bytes.Buffer
	require.NoError(t, graph.RenderSixel(&buf, data, graph.SixelOptions{Width: 200, Height: 120}))
	assert.Contains(t, buf.String(), "\x1bP")
	assert.Contains(t, buf.String(), "a: 5.00")
}

func TestRenderSixelHandlesEmptyAndNonFiniteData(t *testing.T) {
	t.Run("nil data prints a message without Sixel", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, graph.RenderSixel(&buf, nil, graph.DefaultSixelOptions()))
		assert.Equal(t, "No data to display\n", buf.String())
	})

	t.Run("only non-finite values still encodes", func(t *testing.T) {
		t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		data := &graph.ChartData{Series: []graph.Series{{
			Name:   "bad",
			Points: []graph.Point{{Time: t0, Value: math.Inf(1)}, {Time: t0.Add(time.Minute), Value: math.NaN()}},
		}}}

		var buf bytes.Buffer
		require.NoError(t, graph.RenderSixel(&buf, data, graph.SixelOptions{Width: 200, Height: 120}))
		assert.Contains(t, buf.String(), "\x1bP")
	})
}
