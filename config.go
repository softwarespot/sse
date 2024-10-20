package sse

import "time"

// Config defines the configuration settings for the Server-Sent Events (SSE) handler.
type Config[T any] struct {
	// How often to flush the events to the connected clients. Default is 256ms
	FlushFrequency time.Duration

	// How long to wait for all connected client to gracefully close. Default is 30s
	CloseTimeout time.Duration

	// Replay events when a client connects
	Replay struct {
		// How many events to send in chunks, when a client connects. Default is 256 events
		Initial int

		// How many events to keep in memory. Default is 2048 events
		Maximum int

		// How long an event should be kept in memory for. Default is 30s
		Expiry time.Duration
	}

	// Events encoder function, which returns a slice of bytes that will then be converted to a string. Default is json.Marshal()
	Encoder func([]T) ([]byte, error)
}

// NewConfig initializes a configuration instance with reasonable defaults.
func NewConfig[T any]() *Config[T] {
	cfg := &Config[T]{}
	cfg.FlushFrequency = 256 * time.Millisecond
	cfg.CloseTimeout = 30 * time.Second
	cfg.Replay.Initial = 256
	cfg.Replay.Maximum = 2048
	cfg.Replay.Expiry = 30 * time.Second

	// Use the default encoder
	cfg.Encoder = nil

	return cfg
}
