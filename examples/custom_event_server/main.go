package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/softwarespot/sse"
)

type CustomEvent struct {
	ID string `json:"id"`
}

func main() {
	ctx0, cancel0 := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel0()

	ctx1, cancel1 := context.WithTimeout(ctx0, 5*time.Minute)
	defer cancel1()

	// Use the default configuration
	h := sse.New[CustomEvent](nil)
	go func() {
		for {
			evt1 := CustomEvent{
				ID: must(createID[string]()),
			}
			evt2 := CustomEvent{
				ID: must(createID[string]()),
			}
			fmt.Println("sse handler: broadcast event", h.Broadcast(evt1, evt2))
			time.Sleep(64 * time.Millisecond)
		}
	}()

	// Start the server on port "3000" as non-blocking
	go func() {
		http.Handle("/events", h)
		http.ListenAndServe(":3000", nil)
	}()

	// Wait for either a termination signal or timeout of the context
	<-ctx1.Done()

	if err := h.Close(); err != nil {
		fmt.Println("sse handler: server shutdown with error:", err)
		os.Exit(1)
	}
	fmt.Println("sse handler: server shutdown gracefully")
}

// Helpers

func createID[T ~string]() (T, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b[:24]); err != nil {
		return "", fmt.Errorf("creating ID: %w", err)
	}

	// 8 bytes used for the timestamp i.e. 32 - 8 = 24
	binary.BigEndian.PutUint64(b[24:], uint64(time.Now().UnixMilli()))
	return T(hex.EncodeToString(b)), nil
}

func must[T any](res T, err error) T {
	if err != nil {
		panic(err)
	}
	return res
}
