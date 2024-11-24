package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/softwarespot/sse"
)

type PingEvent struct {
	Duration string `json:"duration"`
	Host     string `json:"host"`
}

func main() {
	ctx0, cancel0 := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel0()

	ctx1, cancel1 := context.WithTimeout(ctx0, 5*time.Minute)
	defer cancel1()

	var wg sync.WaitGroup
	s := http.Server{
		Addr: ":3000",
	}

	// Start the server on port "3000" as non-blocking
	go func() {
		var connectedClients atomic.Int64
		http.HandleFunc("/state", func(w http.ResponseWriter, _ *http.Request) {
			res := map[string]any{
				"clients": connectedClients.Load(),
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(res); err != nil {
				fmt.Println("unable to encode the route /state JSON response. Error:", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		})
		http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
			wg.Add(1)
			defer wg.Done()

			connectedClients.Add(1)
			defer connectedClients.Add(-1)

			queryParams := r.URL.Query()
			if !queryParams.Has("host") {
				http.Error(w, "missing host query param", http.StatusBadRequest)
				return
			}
			host := queryParams.Get("host")
			if host == "" {
				http.Error(w, "empty host query param", http.StatusBadRequest)
				return
			}

			// Use the default configuration
			h := sse.New[PingEvent](nil)
			fmt.Println("sse handler: started handler")

			ctxRequest, requestCancel := context.WithCancelCause(r.Context())
			defer requestCancel(nil)

			go monitorPing(ctxRequest, requestCancel, h, host)

			fmt.Println("sse handler: client connected")
			err := h.ServeSSE(w, r.WithContext(ctxRequest))
			fmt.Println("sse handler: client disconnected", err)

			fmt.Println("sse handler: closed handler", h.Close())
		})
		s.ListenAndServe()
	}()

	// Wait for either a termination signal or timeout of the context
	<-ctx1.Done()

	fmt.Println("start server shutdown")
	ctxShutdown, shutdownCancel := context.WithTimeout(ctx0, 5*time.Second)
	defer shutdownCancel()

	err := s.Shutdown(ctxShutdown)
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Println("server shutdown with error:", err)
		os.Exit(1)
	}

	wg.Wait()
	fmt.Println("server shutdown gracefully")
}

func monitorPing(ctx context.Context, cancel context.CancelCauseFunc, h *sse.Handler[PingEvent], host string) {
	defer cancel(nil)

	cmd := exec.CommandContext(ctx, "ping", host)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel(fmt.Errorf("unable to connect to ping stdout pipe: %w", err))
		return
	}
	if err := cmd.Start(); err != nil {
		cancel(fmt.Errorf("unable to start ping command: %w", err))
		return
	}

	fmt.Println("sse handler: strated ping command with PID:", cmd.Process.Pid)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		duration := parsePingDuration(scanner.Text())
		if duration == "" {
			continue
		}

		evt := PingEvent{
			Duration: duration,
			Host:     host,
		}
		if err := h.Broadcast(evt); err != nil {
			cancel(fmt.Errorf("unable to broadcast ping command result: %w", err))
			return
		}
	}
	if err := cmd.Wait(); err != nil {
		cancel(fmt.Errorf("unable to wait for ping command: %w", err))
	}
}

func parsePingDuration(res string) string {
	_, after, ok := strings.Cut(res, "time=")
	if !ok {
		return ""
	}
	return strings.ReplaceAll(after, " ", "")
}
