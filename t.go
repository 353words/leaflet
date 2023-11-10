package main

import (
	"encoding/json"
	"html/template"
	"os"
	"strings"
	"time"
)

type Loc struct {
	Lat  float64
	Lng  float64
	Time time.Time
}

func locJS(loc Loc) string {
	m := map[string]any{
		"lat":  loc.Lat,
		"lng":  loc.Lng,
		"time": loc.Time.Format("15:04"),
	}
	data, _ := json.Marshal(m)
	return string(data)
}

func main() {
	tmpl := `
var points = [
{{ . }}
];
`

	values := []Loc{
		{1, 2, time.Now()},
		{3, 4, time.Now()},
	}

	t := template.Must(template.New("home").Parse(tmpl))
	pts := make([]string, len(values))
	for i, loc := range values {
		pts[i] = locJS(loc)
	}
	data := strings.Join(pts, ",")
	t.Execute(os.Stdout, template.HTML(data))
}
