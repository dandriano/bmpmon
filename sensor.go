package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
	"github.com/go-zeromq/zyre"
)

const (
	ZRESHOUT string = "SHOUT"
	BMPCHAN  string = "BMPCHAN"
)

type sensorResponse struct {
	Timestamp   time.Time     `json:"timestamp"`
	Elapsed     time.Duration `json:"elapsed"`
	Temperature float32       `json:"temperature"`
	Pressure    float32       `json:"pressure"`
	Altitude    float32       `json:"altitude"`
	Error       error         `json:"error"`
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

	return &sensor{i2c: i2c,
		bmpsens: sens}, nil
}

func (s *sensor) Listen(ctx context.Context) {
	var i uint8

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	n := zyre.NewZyre(ctx)
	if err := n.
		//SetInterface("wlan0").
		Start(); err != nil {
		return
	}
	defer n.Stop()

	log.Println("BMPSENSOR:\tOn")
	log.Println("-----------------------")
	log.Printf("BMPSENSOR:\tJoining %s..\n", BMPCHAN)
	n.Join(BMPCHAN)
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-n.Events():
			log.Printf("BMPSENSOR:%s:\tPolling events\n", BMPCHAN)
			if len(n.Peers()) != 0 {
				log.Printf("BMPSENSOR:%s:\tSomeone here, polling events..\n", BMPCHAN)
				if e.Type == ZRESHOUT {
					log.Printf("BMPSENSOR:%s:\tPeek sensor\n", BMPCHAN)
					i++
					st := time.Now()
					temp, errTemp := s.bmpsens.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
					press, errPress := s.bmpsens.ReadPressureMmHg(bsbmp.ACCURACY_STANDARD)
					alt, errAlt := s.bmpsens.ReadAltitude(bsbmp.ACCURACY_STANDARD)
					el := time.Now().Sub(st)

					r, _ := json.Marshal(sensorResponse{
						Timestamp:   time.Now(),
						Temperature: temp,
						Pressure:    press,
						Altitude:    alt,
						Elapsed:     el,
						Error:       errors.Join(errTemp, errPress, errAlt)})

					log.Printf("BMPSENSOR:%s:\tWhisper sensor\n", BMPCHAN)
					n.Whisper(e.PeerUUID, r)
				}
			}
		case <-ticker.C:
			log.Println("-----------------------")
			log.Println("BMPSENSOR:\tNoActivity")
			temp, _ := s.bmpsens.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
			log.Printf("BMPSENSOR:%s:\tSensor heartbeat b%d ch%d temp%f\n",
				BMPCHAN,
				s.i2c.GetBus(),
				s.i2c.GetAddr(),
				temp)
			log.Printf("BMPSENSOR:%s:\tPeers count %d\n", BMPCHAN, len(n.Peers()))
		}
	}
}

func (s *sensor) Close() {
	log.Println("BMPSENSOR:\tOff")
	s.i2c.Close()
}
