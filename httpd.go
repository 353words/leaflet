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

func (a *API) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.log.Error("bad method", "method", r.Method)
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}
	w.Write(indexHTML)
}

func center(gpx GPX) (float64, float64) {
	lat, lng := 0.0, 0.0

	for _, pt := range gpx.Points {
		lat += pt.Lat
		lng += pt.Lng
	}

	size := float64(len(gpx.Points))
	return lat / size, lng / size
}

func mean(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}

func byMinute(points []Point) []Point {
	minute := -1
	var lats [][]float64
	var lngs [][]float64
	var times []time.Time

	// Group by minute
	for _, pt := range points {
		if pt.Time.Minute() != minute { // New minute group
			lngs = append(lngs, []float64{pt.Lng})
			lats = append(lats, []float64{pt.Lat})
			times = append(times, pt.Time)
			minute = pt.Time.Minute()
			continue
		}
		i := len(lats) - 1
		lats[i] = append(lats[i], pt.Lat)
		lngs[i] = append(lngs[i], pt.Lng)
	}

	// Average per minute
	avgs := make([]Point, len(lngs))
	for i := range lngs {
		avgs[i].Time = times[i]
		avgs[i].Lat = mean(lats[i])
		avgs[i].Lng = mean(lngs[i])
	}

	return avgs
}

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
	minPts := byMinute(gpx.Points)
	a.log.Info("minute agg", "count", len(minPts))
	points := make([]map[string]any, len(minPts))
	for i, pt := range minPts {
		points[i] = map[string]any{
			"Lat":  pt.Lat,
			"Lng":  pt.Lng,
			"Time": pt.Time.Format("15:04"), // HH:MM
		}
	}

	clat, clng := center(gpx)
	data := map[string]any{
		"Name":   gpx.Name,
		"Center": map[string]float64{"Lat": clat, "Lng": clng},
		"Points": points,
	}

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
		Addr:    addr,
		Handler: mux,
	}

	log.Info("server starting", "address", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("can't serve", "error", err)
		os.Exit(1)
	}
}
