package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func loadGPX(t *testing.T) io.Reader {
	file, err := os.Open("testdata/Morning_Walk.gpx")
	require.NoError(t, err, "open file")
	t.Cleanup(func() { file.Close() })
	return file
}

func TestParseGPX(t *testing.T) {
	r := loadGPX(t)
	gpx, err := ParseGPX(r)
	require.NoError(t, err, "parse")
	require.Equal(t, "Morning Walk", gpx.Name)
	require.Equal(t, 2600, len(gpx.Points))
}

func Test_meanByMinute(t *testing.T) {
	r := loadGPX(t)
	gpx, err := ParseGPX(r)
	require.NoError(t, err, "parse")

	pts := meanByMinute(gpx.Points)
	require.Equal(t, 54, len(pts))
}
