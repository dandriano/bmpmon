package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/d2r2/go-logger"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

func main() {
	logger.ChangePackageLogLevel("i2c", logger.WarnLevel)
	logger.ChangePackageLogLevel("bsbmp", logger.WarnLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// init BMP sensor via i2c
	// i2cdetect to discover addresses in use
	sens, err := NewSensor(ctx, 0x77, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer sens.Close()

	// init sqlite storage
	st, err := NewStorage("storage.sqlite3", 10)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	// "io"-communication
	io := make(chan sensorResponse)
	go sens.pool(ctx, 15*time.Minute, io)
	go st.serve(ctx, io)

	// some sort of "relay" commands
	peekData := func(count int) []sensorResponse {
		peek, err := sens.Peek()
		if err != nil {
			log.Fatal(err)
		}

		resp, err := st.Fetch(count)
		if err != nil {
			log.Fatal(err)
		}
		return append(resp, peek)
	}

	genChart := func(count int) ([]string, []opts.LineData, []opts.LineData) {
		// generate data for charts
		x := make([]string, 0)
		t := make([]opts.LineData, 0)
		p := make([]opts.LineData, 0)

		for _, re := range peekData(count) {
			x = append(x, re.Timestamp.Format(time.RFC1123))
			t = append(t, opts.LineData{Value: re.Temperature})
			p = append(p, opts.LineData{Value: re.Pressure})
		}

		return x, t, p
	}

	// serve requests: json
	http.HandleFunc("/json/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Content-Type", "application/json")

		parts := strings.Split(r.URL.Path, "/")
		x := 10
		if len(parts) > 3 {
			if x, err = strconv.Atoi(parts[2]); err != nil {
				log.Fatal(err)
			}
		}
		json.NewEncoder(w).Encode(peekData(x))
		log.Printf("BMPMON:\tServed /json addr=%v\tela=%v\n", r.RemoteAddr, time.Since(start))
	})

	// serve requests: chart
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// build charts
		x, t, p := genChart(10)
		temperatureLine := charts.NewLine()
		temperatureLine.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{
				Theme: types.ThemeWesteros}),
			charts.WithTitleOpts(opts.Title{
				Title: "Temperature log",
			}))

		temperatureLine.SetXAxis(x).
			AddSeries("Temperature °C", t).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{
				Color:      "blue",
				Smooth:     true,
				Symbol:     "circle",
				ShowSymbol: true}))

		pressureLine := charts.NewLine()
		pressureLine.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{
				Theme: types.ThemeWesteros}),
			charts.WithTitleOpts(opts.Title{
				Title: "Pressure log",
			}))

		pressureLine.SetXAxis(x).
			AddSeries("Pressure mm Hg", p).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{
				Color:      "red",
				Smooth:     true,
				Symbol:     "circle",
				ShowSymbol: true}))

		// render page
		page := components.NewPage()
		page.PageTitle = "slack@sarpi"
		page.SetLayout(components.PageFlexLayout)
		page.AddCharts(temperatureLine, pressureLine)
		page.Render(w)

		log.Printf("BMPMON:\tServed / addr=%v\tela=%v\n", r.RemoteAddr, time.Since(start))
	})

	log.Println("BMPMON:\tOn")
	log.Println("-----------------------")
	http.ListenAndServe(":80", nil)
}
