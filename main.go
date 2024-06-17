package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/d2r2/go-logger"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	_ "github.com/mattn/go-sqlite3"
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

	// serve requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		peek, err := sens.Peek()
		if err != nil {
			log.Fatal(err)
		}

		resp, err := st.Fetch(10)
		if err != nil {
			log.Fatal(err)
		}
		resp = append(resp, peek)

		// generate data for charts
		x := make([]string, 0)
		t := make([]opts.LineData, 0)
		p := make([]opts.LineData, 0)

		for _, re := range resp {
			x = append(x, re.Timestamp.Format(time.RFC1123))
			t = append(t, opts.LineData{Value: re.Temperature})
			p = append(p, opts.LineData{Value: re.Pressure})
		}

		// build charts
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

		log.Printf("BMPMON:\tServed addr=%v\tela=%v\n", r.RemoteAddr, time.Since(start))
	})

	log.Println("BMPMON:\tOn")
	log.Println("-----------------------")
	http.ListenAndServe(":80", nil)
}
