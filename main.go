package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/d2r2/go-logger"
	"github.com/go-echarts/go-echarts/v2/charts"
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

		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithInitializationOpts(opts.Initialization{
				PageTitle: "slack@sarpi",
				Theme:     types.ThemeWesteros}),
			charts.WithTitleOpts(opts.Title{
				Title:    "Temperature log history",
				Subtitle: "powered by slack@sarpi",
			}))

		x := make([]string, 0)
		t := make([]opts.LineData, 0)
		p := make([]opts.LineData, 0)

		for _, re := range resp {
			x = append(x, re.Timestamp.Format(time.RFC1123))
			t = append(t, opts.LineData{Value: re.Temperature})
			p = append(p, opts.LineData{Value: re.Pressure})
		}

		line.SetXAxis(x).
			AddSeries("Temp C", t).
			AddSeries("Press MmHg", p).
			SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{
				Smooth:     true,
				Symbol:     "circle",
				ShowSymbol: true}))

		line.Render(w)
		// json.NewEncoder(w).Encode(resp)

		log.Printf("BMPMON:\tServed addr=%v\tela=%v\n", r.RemoteAddr, time.Since(start))
	})

	log.Println("BMPMON:\tOn")
	log.Println("-----------------------")
	http.ListenAndServe(":80", nil)
}
