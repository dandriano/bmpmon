package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/d2r2/go-logger"
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
	st, err := NewStorage("db.sql", 5)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	// "io"-communication
	io := make(chan sensorResponse)
	go sens.pool(ctx, 30*time.Minute, io)
	go st.serve(ctx, io)

	// serve requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		resp, err := st.Fetch(4)
		if err != nil {
			log.Fatal(err)
		}
		p, err := sens.Peek()
		if err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(append(resp, p))
		log.Printf("BMPMON:\tServed %v for %v", r.RemoteAddr, time.Since(start))
	})

	log.Println("BMPMON:\tOn")
	http.ListenAndServe(":80", nil)
}
