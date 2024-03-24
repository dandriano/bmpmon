package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
)

// representation of bmp sensor response
type sensorResponse struct {
	Timestamp   time.Time `json:"timestamp"`
	Temperature float32   `json:"temperature"`
	Pressure    float32   `json:"pressure"`
	Altitude    float32   `json:"altitude"`
}

func main() {
	// init BMP sensor via i2c
	// i2cdetect to discover addresses in use
	i2c, err := i2c.NewI2C(0x77, 1)
	if err != nil {
		return
	}
	defer i2c.Close()

	sensor, err := bsbmp.NewBMP(bsbmp.BMP280, i2c)
	if err != nil {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case "GET":
			temp, errTemp := sensor.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
			press, errPress := sensor.ReadPressureMmHg(bsbmp.ACCURACY_STANDARD)
			alt, errAlt := sensor.ReadAltitude(bsbmp.ACCURACY_STANDARD)

			if err := errors.Join(errTemp, errPress, errAlt); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := sensorResponse{
				Timestamp:   time.Now(),
				Temperature: temp,
				Pressure:    press,
				Altitude:    alt,
			}

			writer.WriteHeader(http.StatusOK)
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(response)
		default:
			http.NotFound(writer, request)
		}
	})

	s := &http.Server{
		Handler: mux,
	}
	s.ListenAndServe()
}
