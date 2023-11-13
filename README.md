# Visualizing Map Data with Leaflet JS
+++
title = "Visualizing Map Data with Leaflet JS"
date = "FIXME"
tags = ["golang"]
categories = [ "golang" ]
url = "FIXME"
author = "mikit"
+++

### Overview

I'm close to my goal of hitting 1,000 kilometer of walking this year.
Whenever I try a new route, I record it. The data that comes out of the recording application ([Strava](https://www.strava.com/dashboard) in my case) is in [GPX format](https://en.wikipedia.org/wiki/GPS_Exchange_Format).
Starva does visualize the route, but I like to do it on my own as well.
Which brought me to this blog post about using [Leaflet JS](https://leafletjs.com/) to visualize GPX data.

We're going to write an HTTP server that accepts a GPX file and returns an interactive map showing the points in the GPX.
The map will look like:

![](map.png)

### Raw Data

Let's start by having a look at the GPX file first


**Listing 1: Morning_Walk.gpx**

```html
01 <?xml version="1.0" encoding="UTF-8"?>
02 <gpx creator="StravaGPX Android" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.topografix.com/GPX/1/1 http://www.topografix.com/GPX/1/1/gpx.xsd" version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
03  <metadata>
04   <time>2023-01-03T06:50:42Z</time>
05  </metadata>
06  <trk>
07   <name>Morning Walk</name>
08   <type>walking</type>
09   <trkseg>
10    <trkpt lat="32.5254730" lon="34.9429370">
11     <ele>19.5</ele>
12     <time>2023-01-03T06:50:42Z</time>
13    </trkpt>
14    <trkpt lat="32.5254790" lon="34.9429480">
15     <ele>19.5</ele>
16     <time>2023-01-03T06:50:43Z</time>
17    </trkpt>
...
10410   </trkseg>
10411  </trk>
10412 </gpx>
```

Listing 1 shows a truncated version of `Morning_Walk.gpx`. You can see it's an XML format and the location data is in `trkpt` elements under `trkseg` element.

Let's run a quick query to see how many points are in this file:

**Listing 2: Number of Points**

```bash
$ grep '<trkpt' Morning_Walk.gpx | wc -l
2600
```

In listing 2 you use `grep` and `wc` to find how many points are in the GPX files. You can see it's 2,600 points which is too much to display. You are going to aggregate points by the minute in order to reduce the number of points on the map.

_Note: Parsing XML with grep is not the best option, for this quick and dirty look it OK, but you should probably use tools such as [XMLStarlet](https://xmlstar.sourceforge.net/)_

### Parsing GPX

Let's start by parsing the GPX file using the build-in `encoing/xml` package.

**Listing 3: Parsing GPX**

```go
09 // Point is a point in GPX data.
10 type Point struct {
11     Lat  float64
12     Lng  float64
13     Time time.Time
14 }
15 
16 // GPX is data in GPX file.
17 type GPX struct {
18     Name   string
19     Time   time.Time
20     Points []Point
21 }
22 
23 // ParseGPX parses GPX file, returns GPX.
24 func ParseGPX(r io.Reader) (GPX, error) {
25     var xmlData struct {
26         Meta struct {
27             Time time.Time `xml:"time"`
28         } `xml:"metadata"`
29         Trk struct {
30             Name    string `xml:"name"`
31             Segment struct {
32                 Points []struct {
33                     Lat  float64   `xml:"lat,attr"`
34                     Lon  float64   `xml:"lon,attr"`
35                     Time time.Time `xml:"time"`
36                 } `xml:"trkpt"`
37             } `xml:"trkseg"`
38         } `xml:"trk"`
39     }
40 
41     dec := xml.NewDecoder(r)
42     if err := dec.Decode(&xmlData); err != nil {
43         return GPX{}, err
44     }
45 
46     gpx := GPX{
47         Name:   xmlData.Trk.Name,
48         Time:   xmlData.Meta.Time,
49         Points: make([]Point, len(xmlData.Trk.Segment.Points)),
50     }
51 
52     for i, pt := range xmlData.Trk.Segment.Points {
53         gpx.Points[i].Lat = pt.Lat
54         gpx.Points[i].Lng = pt.Lon
55         gpx.Points[i].Time = pt.Time
56     }
57 
58     return gpx, nil
59 }
```

Listing 3 shows how to parse a GPX file. On lines 10 and 21 you define the `Point` and `GPX` structs. They are the types returned from parsing. As a general rule, don't expose the internal data structures (e.g. the one in the XML) to your API.
For example, GPX calls longitude `lon` while leaflet uses `lng`.
On line 24 you define the `ParseGPX` function that accepts an `io.Reader`.
On lines 25-39 you define an anonymous struct that corresponds to the structure of the GPX xml. There is no need to model the whole structure of the XML, only the elements you are interested in.
On line 33 and 34 you specify the `Lat` and `Lng` are not XML elements but attributes using `,attr` in the field tag.
On lines 41 to 44 you use an XML decoder to parse the data into the `xmlData` struct.
On lines 46 to 56 you transform data in `xmlData` to the API level `GPX` type.
Finally, on line 58 you return the GPX.

### Data Aggregation

Since there are too many points to display on the map, you are going to aggregate the points by minute.
This is similar to SQL [`GROUP BY`](https://en.wikipedia.org/wiki/Group_by_(SQL)) where you first group rows to buckets depending on a key (the time rounded to a minute in our case) and then run an aggregation on the values in each bucket (mean in our case). The SQL code (for SQLite3) can be something like:

```sql
SELECT
    strftime('%H:%M', time),
    AVG(lat),
    AVG(lng)
FROM pts
    GROUP BY strftime('%H:%M', time)
;
```

**Listing 4: Aggregation**

```go
61 // roundToMinute rounds time to minute granularity.
62 func roundToMinute(t time.Time) time.Time {
63     year, month, day := t.Year(), t.Month(), t.Day()
64     hour, minute := t.Hour(), t.Minute()
65 
66     return time.Date(year, month, day, hour, minute, 0, 0, t.Location())
67 }
68 
69 // meanByMinute aggregates points by the minute.
70 func meanByMinute(points []Point) []Point {
71     // Aggregate columns
72     lats := make(map[time.Time][]float64)
73     lngs := make(map[time.Time][]float64)
74 
75     // Group by minute
76     for _, pt := range points {
77         key := roundToMinute(pt.Time)
78         lats[key] = append(lats[key], pt.Lat)
79         lngs[key] = append(lngs[key], pt.Lng)
80     }
81 
82     // Average per minute
83     avgs := make([]Point, len(lngs))
84     i := 0
85     for time, lats := range lats {
86         avgs[i].Time = time
87         avgs[i].Lat = mean(lats)
88         avgs[i].Lng = mean(lngs[time])
89         i++
90     }
91 
92     return avgs
93 }
```

Listing 4 shows `meanByMinute` that aggregates points by minute.
On lines 72 and 73 we define the aggregation columns.
On lines 76-80 we group points by minute. 
On lines 83-80 we create new slice of points where each point has the group time and the average of latitude and longitude.

### Map HTML Template

You are going to use `html/template` to render the map. Most of the HTML is static and you'll generate the title, data and list of points dynamically.

**Listing 5: Map HTML Template**

```html
01 <!doctype html>
02 <html>
03   <head>
04     <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" 
05       integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65"
06       crossorigin="anonymous">
07     <meta name="viewport" content="width=device-width, initial-scale=1">
08     <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"
09       integrity="sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY="
10       crossorigin=""/>
11     <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"
12      integrity="sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo="
13      crossorigin=""></script>
14   </head>
15   <body>
16     <div class="container">
17       <div class="row text-center">
18     <h1 class="alert alert-primary" role="alert">GPX File Viewer</h1>
19     <h3 class="alert alert-secondary" role="alert">{{ .Name }} on {{ .Date }}</h1>
20       </div>
21       <div class="row">
22     <div class="col">
23       <div id="map" style="height: 600px; border: 1px solid black;"></div>
24     </div>
25       </div>
26     </div>
27     <script>
28       var points = [
29     {{- range $idx, $pt := .Points }}
30     {{ if $idx }},{{ end -}}
31     { "lat": {{ $pt.Lat }}, "lng": {{ $pt.Lng -}}, "time": {{ $pt.Time }} }
32     {{- end }}
33       ];
34 
35       function on_loaded() {
36     var map = L.map('map').setView([{{ .Center.Lat }}, {{ .Center.Lng }}], 15);
37     L.tileLayer(
38       'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
39       {
40         maxZoom: 19,
41         attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
42       }
43     ).addTo(map);
44     points.forEach((pt) => {
45       let circle = L.circle(
46         [pt.lat, pt.lng],
47         {
48           color: 'red',
49           radius: 20
50         }).addTo(map);
51       circle.bindPopup(pt.time);
52     });
53       }
54 
55       document.addEventListener('DOMContentLoaded', on_loaded);
56     </script>
57   </body>
58 </html>

```

Listing 5 shows the map HTML template file.
On lines 04-11 we import [bootstrap](https://getbootstrap.com/) for a nice UI and also the leafletjs CSS and JS files.
On line 19 we use the template to set the name and date of the GPX file.
On lines 29-32 we generate a JavaScript array with the points from the input.
On lines 35-53 we write JavaScript code to generate the map. On line 36 we create the map with a center location and set the zoom level to 20. On lines 37-43 we load the tiles from [OpenStreetMap](https://www.openstreetmap.org/). On lines 44-53 we iterate over the points, adding them to the map on lines 45-50 as a red circle and setting the tooltip to be the hour on line 51.

### Map HTTP Handler

**Listing 6: Map HTTP Handler**

```go
61 // mapHandler gets GPX file via HTML form and return map from mapTemplate.
62 func (a *API) mapHandler(w http.ResponseWriter, r *http.Request) {
63     a.log.Info("map called", "remote", r.RemoteAddr)
64     if r.Method != http.MethodPost {
65         a.log.Error("bad method", "method", r.Method)
66         http.Error(w, "bad method", http.StatusMethodNotAllowed)
67         return
68     }
69 
70     if err := r.ParseMultipartForm(1 << 20); err != nil {
71         a.log.Error("bad form", "error", err)
72         http.Error(w, "bad form", http.StatusBadRequest)
73         return
74     }
75 
76     file, _, err := r.FormFile("gpx")
77     if err != nil {
78         a.log.Error("missing gpx file", "error", err)
79         http.Error(w, "missing gpx file", http.StatusBadRequest)
80         return
81     }
82 
83     gpx, err := ParseGPX(file)
84     if err != nil {
85         a.log.Error("bad gpx", "error", err)
86         http.Error(w, "bad gpx", http.StatusBadRequest)
87         return
88     }
89 
90     a.log.Info("gpx parsed", "name", gpx.Name, "count", len(gpx.Points))
91     meanPts := meanByMinute(gpx.Points)
92     a.log.Info("minute agg", "count", len(meanPts))
93 
94     // Data for template
95     points := make([]map[string]any, len(meanPts))
96     for i, pt := range meanPts {
97         points[i] = map[string]any{
98             "Lat":  pt.Lat,
99             "Lng":  pt.Lng,
100             "Time": pt.Time.Format("15:04"), // HH:MM
101         }
102     }
103 
104     clat, clng := center(gpx.Points)
105     data := map[string]any{
106         "Name":   gpx.Name,
107         "Date":   gpx.Time.Format(time.DateOnly),
108         "Center": map[string]float64{"Lat": clat, "Lng": clng},
109         "Points": points,
110     }
111 
112     w.Header().Set("content-type", "text/html")
113     if err := mapTemplate.Execute(w, data); err != nil {
114         a.log.Error("can't execute template", "error", err)
115     }
116 }
```

Listing 6 shows the map HTTP handler.
On lines 70-74 you parse the HTTP form and get the GPX file from the form.
On line 83-88 you parse the GPX and aggregate the points.
On lines 94-110 you generate the data for the template. On lines 95-102 you create the slice of points. On line 104 you calculate the center of the map and on lines 105-110 you create the `data` map that contains all the elements.
One line 112 you set the content type and on lines 113-115 you execute the template with the data.

_Note: The `center` function is not shown here. You can view it in the GitHub repository._

### Starting The Server

**Listing 7: HTTP Handler**

```go
12 var (
13     //go:embed static/index.html
14     indexHTML []byte
15 
16     //go:embed static/map.html
17     mapHTML string
18 
19     mapTemplate *template.Template
20 )
21 
22 type API struct {
23     log *slog.Logger
24 }
...
117 
118 func main() {
119     log := slog.New(slog.NewTextHandler(os.Stdout, nil))
120     tmpl, err := template.New("map").Parse(mapHTML)
121     if err != nil {
122         log.Error("can't parse map HTML", "error", err)
123         os.Exit(1)
124     }
125     mapTemplate = tmpl
126 
127     api := API{
128         log: log,
129     }
130 
131     mux := http.NewServeMux()
132     mux.HandleFunc("/", api.indexHandler)
133     mux.HandleFunc("/map", api.mapHandler)
134 
135     addr := ":8080"
136     srv := http.Server{
137         Addr:              addr,
138         Handler:           mux,
139         ReadHeaderTimeout: time.Second,
140     }
141 
142     log.Info("server starting", "address", addr)
143     if err := srv.ListenAndServe(); err != nil {
144         log.Error("can't serve", "error", err)
145         os.Exit(1)
146     }
147 }
```

Listing 7 shows the how you run the HTTP server.
On lines 12-20 you embed the template in the executable using the `embed` package.
On lines 22-24 you define the API struct.
On line 1 you create a logger from the `log/slog` package.
On lines 120-125 you parse the map HTML template and set the package level `mapTemplate` variable.
On lines 127-129 you create and API and on lines 131-133 you set the routing.
On lines 135-140 you create an HTTP server and on lines 142-146 you run it.


### Conclusion

Leaflet JS is a great library for map visualization, it uses OpenStreetMap for which has many layers of detailed data.
If find it very cool that it only took about 270 lines of Go and JavaScript code to generate an interactive map from raw GPX data.

How do you visualize map data, I'd love to hear from you at miki@ardanlabs.com
