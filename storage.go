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
    ?, ?, ?, ?
);
`
	fetch = `
SELECT time, temperature, altitude, pressure FROM sensor_log 
ORDER BY time DESC 
LIMIT ?;
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
	db     *sql.DB
	insert *sql.Stmt
	buffer []sensorResponse
}

func NewStorage(connect string, bufferSize int) (*storage, error) {
	conn, err := sql.Open("sqlite3", connect)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	if _, err := conn.Exec(schema); err != nil {
		return nil, err
	}

	insertStmt, err := conn.Prepare(insert)
	if err != nil {
		return nil, err
	}

	return &storage{
		db:     conn,
		insert: insertStmt,
		buffer: make([]sensorResponse, 0, bufferSize),
	}, nil
}

func (s *storage) Add(log sensorResponse) error {
	if len(s.buffer) >= cap(s.buffer) {
		return errors.New("BMPSTORAGE:\tBuffer is full")
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
	if last <= len(s.buffer) {
		return s.buffer[len(s.buffer)-last:], nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Println("BMPSTORAGE:\tFetch failed:", r)
		}
	}()
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

	return res, rows.Err()
}

func (s *storage) Flush() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Println("BMPSTORAGE:\tFlush failed due to panic:", r)
		}
	}()

	for _, record := range s.buffer {
		if _, err := tx.Stmt(s.insert).Exec(record.Timestamp, record.Temperature, record.Altitude, record.Pressure); err != nil {
			tx.Rollback()
			log.Println("BMPSTORAGE:\tFlush failed on record:", record, "Error:", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Println("BMPSTORAGE:\tFlush failed to commit:", err)
		return err
	}

	s.buffer = s.buffer[:0]

	return nil
}

func (s *storage) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}

	if closeErr := s.insert.Close(); closeErr != nil {
		return closeErr
	}

	return s.db.Close()
}

func (s *storage) serve(ctx context.Context, input <-chan sensorResponse) {
	log.Println("BMPSTORAGE:\tOn")
	log.Println("-----------------------")
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-input:
			log.Println("BMPSTORAGE:\tServing response")
			if err := s.Add(r); err != nil {
				log.Fatal(err)
			}
		}
	}
}
