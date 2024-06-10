package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/d2r2/go-logger"
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		resp, err := sens.Peek()
		if err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(resp)
	})

	http.ListenAndServe(":80", nil)
}
