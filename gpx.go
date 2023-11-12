package main

import (
	"encoding/xml"
	"io"
	"time"
)

// Point is a point in GPX data.
type Point struct {
	Lat  float64
	Lng  float64
	Time time.Time
}

// GPX is data in GPX file.
type GPX struct {
	Name   string
	Time   time.Time
	Points []Point
}

// ParseGPX parses GPX file, returns GPX.
func ParseGPX(r io.Reader) (GPX, error) {
	var xmlData struct {
		Meta struct {
			Time time.Time `xml:"time"`
		} `xml:"metadata"`
		Trk struct {
			Name    string `xml:"name"`
			Segment struct {
				Points []struct {
					Lat  float64   `xml:"lat,attr"`
					Lon  float64   `xml:"lon,attr"`
					Time time.Time `xml:"time"`
				} `xml:"trkpt"`
			} `xml:"trkseg"`
		} `xml:"trk"`
	}

	dec := xml.NewDecoder(r)
	if err := dec.Decode(&xmlData); err != nil {
		return GPX{}, err
	}

	gpx := GPX{
		Name:   xmlData.Trk.Name,
		Time:   xmlData.Meta.Time,
		Points: make([]Point, len(xmlData.Trk.Segment.Points)),
	}

	for i, pt := range xmlData.Trk.Segment.Points {
		gpx.Points[i].Lat = pt.Lat
		gpx.Points[i].Lng = pt.Lon
		gpx.Points[i].Time = pt.Time
	}

	return gpx, nil
}

// aggByMinute aggregates points by the minute.
func aggByMinute(points []Point) []Point {
	minute := -1
	// Aggregate columns
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
