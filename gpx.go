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

// roundToMinute rounds time to minute granularity.
func roundToMinute(t time.Time) time.Time {
	year, month, day := t.Year(), t.Month(), t.Day()
	hour, minute := t.Hour(), t.Minute()

	return time.Date(year, month, day, hour, minute, 0, 0, t.Location())
}

// meanByMinute aggregates points by the minute.
func meanByMinute(points []Point) []Point {
	// Aggregate columns
	lats := make(map[time.Time][]float64)
	lngs := make(map[time.Time][]float64)

	// Group by minute
	for _, pt := range points {
		key := roundToMinute(pt.Time)
		lats[key] = append(lats[key], pt.Lat)
		lngs[key] = append(lngs[key], pt.Lng)
	}

	// Average per minute
	avgs := make([]Point, len(lngs))
	i := 0
	for time, lats := range lats {
		avgs[i].Time = time
		avgs[i].Lat = mean(lats)
		avgs[i].Lng = mean(lngs[time])
		i++
	}

	return avgs
}
