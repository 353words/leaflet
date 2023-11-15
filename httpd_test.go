package main

import (
	"bytes"
	"html/template"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_mapHandler(t *testing.T) {
	// error: no multipart boundary param in Content-Type
	//t.Skip("FIXME")
	file := loadGPX(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("gpx", "data.gpx")
	require.NoError(t, err, "create from file")
	_, err = io.Copy(part, file)
	require.NoError(t, err, "copy data")
	mw.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/map", &buf)
	r.Header.Add("Content-Type", mw.FormDataContentType())

	tmpl, err := template.New("map").Parse(mapHTML)
	require.NoError(t, err, "template")
	mapTemplate = tmpl

	api := API{
		log: slog.Default(),
	}

	api.mapHandler(w, r)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read body")
	require.True(t, bytes.Contains(data, []byte("var points = [")))
}
