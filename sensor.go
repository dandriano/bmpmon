package main

import (
	"context"
	"log"
	"time"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
)

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

	res := &sensor{i2c: i2c, bmpsens: sens}

	return res, nil
}

func (s *sensor) pool(ctx context.Context, interval time.Duration, o chan<- sensorResponse) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	f := func() {
		r, err := s.Peek()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("BMPSENSOR:\tPeeked sensor temp=%v\tela=%v\n", r.Temperature, r.Elapsed)
		o <- r
	}

	log.Println("BMPSENSOR:\tOn")
	log.Println("-----------------------")
	f()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f()
		}
	}
}

func (s *sensor) Peek() (sensorResponse, error) {
	st := time.Now()
	r := sensorResponse{}
	temp, err := s.bmpsens.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return r, err
	}
	press, err := s.bmpsens.ReadPressureMmHg(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return r, err
	}
	alt, err := s.bmpsens.ReadAltitude(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return r, err
	}
	r.Timestamp = time.Now()
	r.Altitude = alt
	r.Temperature = temp
	r.Pressure = press
	r.Elapsed = time.Since(st)

	return r, nil
}

func (s *sensor) Close() {
	log.Println("BMPSENSOR:\tOff")
	s.i2c.Close()
}
