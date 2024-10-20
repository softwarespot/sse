package main

import (
	"context"
	"crypto/rand"
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
	ctx, _ := signalTrap(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Use the default configuration
	h := sse.New[CustomEvent](nil)
	go func() {
		for {
			id := must(generateID[string]())
			evt := CustomEvent{
				ID: id,
			}
			fmt.Println("sse handler: broadcast event", h.Broadcast(evt))
			time.Sleep(64 * time.Millisecond)
		}
	}()

	// Start the server on port "3000" as non-blocking
	go func() {
		http.Handle("/events", h)
		http.ListenAndServe(":3000", nil)
	}()

	// Wait for either a termination signal or timeout of the context
	<-ctx.Done()

	if err := h.Close(); err != nil {
		fmt.Println("sse handler: server shutdown with error:", err)
		os.Exit(1)
	}
	fmt.Println("sse handler: server shutdown gracefully")
}

// Helpers

func generateID[T ~string]() (T, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("creating a new ID: %w", err)
	}
	return T(fmt.Sprintf("%x-%d", b, time.Now().UnixMilli())), nil
}

func must[T any](res T, err error) T {
	if err != nil {
		panic(err)
	}
	return res
}

func signalTrap(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, sig...)
		select {
		case <-ctx.Done():
		case <-signals:
		}
		cancel()
	}()
	return ctx, cancel
}
