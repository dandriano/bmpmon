package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	// _ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	schema = `
CREATE TABLE IF NOT EXISTS sensor_log (
    time TIMESTAMP NOT NULL,
    temperature FLOAT,
    altitude FLOAT,
    pressure FLOAT
);

CREATE INDEX IF NOT EXISTS sensor_time ON sensor_log(time);
`
	insert = `
INSERT INTO sensor_log (
    time, temperature, altitude, pressure
)
VALUES (
    $1, $2, $3, $4
);
`
	fetch = `
SELECT time, temperature, altitude, pressure FROM sensor_log 
ORDER BY time DESC 
LIMIT $1;
`
)

type sensorResponse struct {
	Timestamp   time.Time     `json:"time"`
	Elapsed     time.Duration `json:"elapsed"`
	Temperature float32       `json:"temperature"`
	Pressure    float32       `json:"pressure"`
	Altitude    float32       `json:"altitude"`
}

type storage struct {
	sql    *sql.DB
	ins    *sql.Stmt
	buffer []sensorResponse
}

func NewStorage(connect string, bsize int) (*storage, error) {
	// or remote postgres
	conn, err := sql.Open("sqlite3", connect)
	if err != nil {
		return nil, err
	}

	err = conn.Ping()
	if err != nil {
		panic(err)
	}

	if _, err = conn.Exec(schema); err != nil {
		return nil, err
	}

	ins, err := conn.Prepare(insert)
	if err != nil {
		return nil, err
	}

	storage := storage{
		sql:    conn,
		ins:    ins,
		buffer: make([]sensorResponse, 0, bsize),
	}
	return &storage, nil
}

func (s *storage) Add(log sensorResponse) error {
	if len(s.buffer) == cap(s.buffer) {
		return errors.New("buffer is full")
	}

	s.buffer = append(s.buffer, log)
	if len(s.buffer) == cap(s.buffer) {
		if err := s.Flush(); err != nil {
			return err
		}
	}

	return nil
}

func (s *storage) Fetch(last int) ([]sensorResponse, error) {
	if len(s.buffer) >= last {
		return s.buffer[last:], nil
	}

	tx, err := s.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(fetch, last-len(s.buffer))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []sensorResponse
	for rows.Next() {
		var r sensorResponse
		if err := rows.Scan(&r.Timestamp, &r.Temperature, &r.Altitude, &r.Pressure); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	res = append(res, s.buffer...)

	return res, nil
}

func (s *storage) Flush() error {
	tx, err := s.sql.Begin()
	if err != nil {
		return err
	}

	for _, record := range s.buffer {
		_, err := tx.Stmt(s.ins).Exec(record.Timestamp, record.Temperature, record.Altitude, record.Pressure)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	s.buffer = s.buffer[:0]

	return tx.Commit()
}

func (s *storage) Close() error {
	defer func() {
		s.ins.Close()
		s.sql.Close()
	}()

	if err := s.Flush(); err != nil {
		return err
	}

	return nil
}

func (s *storage) serve(ctx context.Context, i <-chan sensorResponse) {
	log.Println("BMPSTORAGE:\tOn")
	log.Println("-----------------------")
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-i:
			log.Println("BMPSTORAGE:\tServing response")
			if err := s.Add(r); err != nil {
				log.Fatal(err)
			}
		}
	}
}
