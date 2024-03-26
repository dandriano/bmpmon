package main

import (
	"context"
	"log"

	"github.com/d2r2/go-logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// init BMP sensor via i2c
	// i2cdetect to discover addresses in use
	sens, err := NewSensor(ctx, 0x77, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer sens.Close()

	logger.ChangePackageLogLevel("i2c", logger.WarnLevel)
	logger.ChangePackageLogLevel("bsbmp", logger.WarnLevel)

	sens.Listen(ctx)
}
