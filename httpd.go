package main

import (
	_ "embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var (
	//go:embed static/index.html
	indexHTML []byte

	//go:embed static/map.html
	mapHTML string

	mapTemplate *template.Template
)

type API struct {
	log *slog.Logger
}

// indexHTML returns the index HTML.
func (a *API) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.log.Error("bad method", "method", r.Method)
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("content-type", "text/html")
	if _, err := w.Write(indexHTML); err != nil {
		a.log.Error("can't write", "error", err)
	}
}

// Center returns the center point (lat, lng) of gpx points
func center(points []Point) (float64, float64) {
	lat, lng := 0.0, 0.0

	for _, pt := range points {
		lat += pt.Lat
		lng += pt.Lng
	}

	size := float64(len(points))
	return lat / size, lng / size
}

// mean returns the mean of values.
func mean(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}

// mapHandler gets GPX file via HTML form and return map from mapTemplate.
func (a *API) mapHandler(w http.ResponseWriter, r *http.Request) {
	a.log.Info("map called", "remote", r.RemoteAddr)
	if r.Method != http.MethodPost {
		a.log.Error("bad method", "method", r.Method)
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(1 << 20); err != nil {
		a.log.Error("bad form", "error", err)
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("gpx")
	if err != nil {
		a.log.Error("missing gpx file", "error", err)
		http.Error(w, "missing gpx file", http.StatusBadRequest)
		return
	}

	gpx, err := ParseGPX(file)
	if err != nil {
		a.log.Error("bad gpx", "error", err)
		http.Error(w, "bad gpx", http.StatusBadRequest)
		return
	}

	a.log.Info("gpx parsed", "name", gpx.Name, "count", len(gpx.Points))
	meanPts := meanByMinute(gpx.Points)
	a.log.Info("minute agg", "count", len(meanPts))

	// Data for template
	points := make([]map[string]any, len(meanPts))
	for i, pt := range meanPts {
		points[i] = map[string]any{
			"Lat":  pt.Lat,
			"Lng":  pt.Lng,
			"Time": pt.Time.Format("15:04"), // HH:MM
		}
	}

	clat, clng := center(gpx.Points)
	data := map[string]any{
		"Name":   gpx.Name,
		"Date":   gpx.Time.Format(time.DateOnly),
		"Center": map[string]float64{"Lat": clat, "Lng": clng},
		"Points": points,
	}

	w.Header().Set("content-type", "text/html")
	if err := mapTemplate.Execute(w, data); err != nil {
		a.log.Error("can't execute template", "error", err)
	}
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpl, err := template.New("map").Parse(mapHTML)
	if err != nil {
		log.Error("can't parse map HTML", "error", err)
		os.Exit(1)
	}
	mapTemplate = tmpl

	api := API{
		log: log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.indexHandler)
	mux.HandleFunc("/map", api.mapHandler)

	addr := ":8080"
	srv := http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}

	log.Info("server starting", "address", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("can't serve", "error", err)
		os.Exit(1)
	}
}
