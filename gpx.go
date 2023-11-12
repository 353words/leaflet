package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"
)

type Point struct {
	Lat  float64
	Lng  float64
	Time time.Time
}

type GPX struct {
	Name   string
	Points []Point
}

func ParseGPX(r io.Reader) (GPX, error) {
	var xmlData struct {
		Trk struct {
			Name    string `xml:"name"`
			Segment struct {
				Points []struct {
					Lat  float64 `xml:"lat,attr"`
					Lon  float64 `xml:"lon,attr"`
					Time string  `xml:"time"`
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
		Points: make([]Point, len(xmlData.Trk.Segment.Points)),
	}
	for i, pt := range xmlData.Trk.Segment.Points {
		t, err := time.Parse(time.RFC3339, pt.Time)
		if err != nil {
			return GPX{}, fmt.Errorf("Point %d: bad time - %s", i, err)
		}
		gpx.Points[i].Lat = pt.Lat
		gpx.Points[i].Lng = pt.Lon
		gpx.Points[i].Time = t
	}

	return gpx, nil
}
