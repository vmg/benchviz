package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Dataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

type ChartData struct {
	Labels   []string   `json:"labels"`
	Datasets []*Dataset `json:"datasets"`
}

type Scale struct {
	Display bool   `json:"display"`
	Type    string `json:"type"`
}

type Chart struct {
	Type    string    `json:"type"`
	Data    ChartData `json:"data"`
	Options struct {
		IndexAxis  string `json:"indexAxis,omitempty"`
		Responsive bool   `json:"responsive"`
		Title      struct {
			Display bool   `json:"display"`
			Text    string `json:"text"`
		} `json:"title"`
		Scales struct {
			XAxes []Scale `json:"xAxes,omitempty"`
			YAxes []Scale `json:"yAxes,omitempty"`
		} `json:"scales"`
	} `json:"options"`
}

type Request struct {
	Format          string `json:"format,omitempty"`
	Chart           Chart  `json:"chart"`
	Version         string `json:"version,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	chunks := bytes.Split(input, []byte{'\n', '\n'})

	for i, chunk := range chunks {
		lines := bytes.Split(chunk, []byte{'\n'})
		if i == 0 {
			lines = lines[4:]
		}

		header := lines[0]
		lines = lines[1 : len(lines)-1]

		if i == len(chunks)-1 {
			lines = lines[:len(lines)-1]
		}

		var buf bytes.Buffer
		for i, line := range lines {
			if i > 0 {
				buf.WriteByte('\n')
			}
			buf.Write(line)
		}

		records, err := csv.NewReader(&buf).ReadAll()
		if err != nil {
			log.Fatal(err)
		}

		var chart Chart
		chart.Type = "horizontalBar"
		chart.Options.Title.Display = true
		chart.Options.Title.Text = records[0][1]

		if i > 0 {
			chart.Options.Scales.XAxes = append(chart.Options.Scales.XAxes, Scale{
				Display: true,
				Type:    "logarithmic",
			})
		}

		for _, label := range strings.Split(string(header), ",") {
			if label == "" {
				continue
			}
			chart.Data.Datasets = append(chart.Data.Datasets, &Dataset{
				Label: label,
			})
		}

		for _, r := range records[1:] {
			benchname := strings.Split(r[0], "/")
			fname := strings.Join(benchname[:len(benchname)-1], "/")
			if len(fname) > 128 {
				fname = fname[:128] + "..."
			}
			chart.Data.Labels = append(chart.Data.Labels, fname)

			for i, dset := range chart.Data.Datasets {
				pos := 1
				if i > 0 {
					pos = (i * 4) - 1
				}

				v, err := strconv.ParseFloat(r[pos], 64)
				if err != nil {
					log.Fatal(err)
				}

				dset.Data = append(dset.Data, v)
			}
		}

		var payload bytes.Buffer
		var request = Request{
			Chart:           chart,
			Format:          "png",
			BackgroundColor: "#ffffff",
		}

		enc := json.NewEncoder(&payload)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		enc.Encode(&request)

		log.Printf("%s", payload.String())

		req, err := http.Post("https://quickchart.io/chart", "application/json", &payload)
		if err != nil {
			log.Fatal(err)
		}

		f, err := os.Create(fmt.Sprintf("%d.png", i))
		if err != nil {
			log.Fatal(err)
		}

		_, _ = io.Copy(f, req.Body)
		req.Body.Close()
		f.Close()
	}
}
