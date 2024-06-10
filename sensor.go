package main

import (
	"context"
	"log"
	"time"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
)

type sensorResponse struct {
	Timestamp   time.Time     `json:"timestamp"`
	Elapsed     time.Duration `json:"elapsed"`
	Temperature float32       `json:"temperature"`
	Pressure    float32       `json:"pressure"`
	Altitude    float32       `json:"altitude"`
}

type sensor struct {
	i2c     *i2c.I2C
	bmpsens *bsbmp.BMP
}

func NewSensor(ctx context.Context, addr uint8, bus int) (*sensor, error) {
	i2c, err := i2c.NewI2C(addr, bus)
	if err != nil {
		return nil, err
	}

	sens, err := bsbmp.NewBMP(bsbmp.BMP280, i2c)
	if err != nil {
		return nil, err
	}

	res := sensor{i2c: i2c, bmpsens: sens}
	go res.pool(ctx)
	return &res, nil
}

func (s *sensor) pool(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("BMPSENSOR:\tOn")
	log.Println("-----------------------")
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, _ := s.Peek()
			log.Printf("BMPSENSOR:\tSensor heartbeat temp%f\n", resp.Temperature)
		}
	}
}

func (s *sensor) Peek() (*sensorResponse, error) {
	log.Print("BMPSENSOR:\tPeek sensor\n")
	st := time.Now()
	temp, err := s.bmpsens.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}
	press, err := s.bmpsens.ReadPressureMmHg(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}
	alt, err := s.bmpsens.ReadAltitude(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}
	el := time.Now().Sub(st)
	log.Printf("BMPSENSOR:\tPeeked sensor for %v\n", el)

	return &sensorResponse{Timestamp: time.Now(),
		Temperature: temp,
		Pressure:    press,
		Altitude:    alt,
		Elapsed:     el}, nil
}

func (s *sensor) Close() {
	log.Println("BMPSENSOR:\tOff")
	s.i2c.Close()
}
