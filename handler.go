package sse

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"time"

	"github.com/softwarespot/replay"
)

type empty struct{}

// Handler is a generic Server-Sent Events (SSE) handler.
type Handler[T any] struct {
	cfg *Config[T]

	closingCh  chan empty
	completeCh chan empty

	clientRegisterCh   chan chan []T
	clientUnregisterCh chan chan []T
	clientEvtsChs      map[chan []T]empty

	evtsReplay *replay.Replay[T]
	evtsCh     chan T

	evtsEncoder func([]T) ([]byte, error)
}

// New initializes a Server-Sent Events (SSE) handler, with an optional configuration.
// If the provided configuration is nil, it will use default settings.
func New[T any](cfg *Config[T]) *Handler[T] {
	if cfg == nil {
		cfg = NewConfig[T]()
	}
	h := &Handler[T]{
		cfg: cfg,

		closingCh:  make(chan empty),
		completeCh: make(chan empty),

		clientRegisterCh:   make(chan chan []T),
		clientUnregisterCh: make(chan chan []T),
		clientEvtsChs:      map[chan []T]empty{},

		evtsReplay: replay.New[T](cfg.Replay.Maximum, cfg.Replay.Expiry),
		evtsCh:     make(chan T),

		evtsEncoder: defaultEventsEncoder[T],
	}
	if h.cfg.Encoder != nil {
		h.evtsEncoder = h.cfg.Encoder
	}

	go h.start()

	return h
}

func (h *Handler[T]) start() {
	flushTicker := time.NewTicker(h.cfg.FlushFrequency)
	defer flushTicker.Stop()

	var (
		isClosing bool
		cleanup   = func() bool {
			isCleanable := isClosing && len(h.clientEvtsChs) == 0
			if !isCleanable {
				return false
			}

			close(h.clientRegisterCh)
			close(h.clientUnregisterCh)
			close(h.evtsCh)
			close(h.completeCh)
			return true
		}
		evts []T
	)
	for {
		select {
		case <-h.closingCh:
			isClosing = true
			if cleanup() {
				return
			}
		case clientEvtsCh := <-h.clientRegisterCh:
			h.clientEvtsChs[clientEvtsCh] = empty{}
			for evts := range h.replayedEvents() {
				clientEvtsCh <- evts
			}
		case clientEvtsCh := <-h.clientUnregisterCh:
			close(clientEvtsCh)
			delete(h.clientEvtsChs, clientEvtsCh)

			if cleanup() {
				return
			}
		case evt := <-h.evtsCh:
			evts = append(evts, evt)
		case <-flushTicker.C:
			if len(evts) == 0 {
				break
			}
			for clientEvtsCh := range h.clientEvtsChs {
				clientEvtsCh <- evts
			}
			for _, evt := range evts {
				h.evtsReplay.Add(evt)
			}
			evts = nil
		}
	}
}

// Close closes the Server-Sent Events (SSE) handler.
// It waits for all clients to complete/close, with a timeout defined in the configuration.
func (h *Handler[T]) Close() error {
	if h.isClosing() {
		return errors.New("sse-handler: handler is closed")
	}

	h.closingCh <- empty{}
	close(h.closingCh)

	// Wait for all clients to complete/close with a timeout
	select {
	case <-h.completeCh:
		h.evtsReplay.Clear()
		return nil
	case <-time.After(h.cfg.CloseTimeout):
		return errors.New("sse-handler: timeout waiting for clients to close")
	}
}

func (h *Handler[T]) isClosing() bool {
	select {
	case <-h.closingCh:
		return true
	case <-h.completeCh:
		return true
	default:
		return false
	}
}

// ServeSSE serves the Server-Sent Events (SSE) to the HTTP response writer.
// It sets the appropriate headers and streams events to the client until the connection is closed.
func (h *Handler[T]) ServeSSE(w http.ResponseWriter, r *http.Request) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("sse-handler: not supported")
	}
	if h.isClosing() {
		return errors.New("sse-handler: handler is closed")
	}

	hdrs := w.Header()
	hdrs.Set("Content-Type", "text/event-stream")
	hdrs.Set("Cache-Control", "no-cache")
	hdrs.Set("Connection", "keep-alive")

	clientEvtsCh := h.register()
	defer h.unregister(clientEvtsCh)

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-h.closingCh:
			return nil
		case evts := <-clientEvtsCh:
			data, err := h.evtsEncoder(evts)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return fmt.Errorf("sse-handler: unable to write events: %w", err)
			}
			flusher.Flush()
		}
	}
}

// ServeHTTP implements the http.Handler interface for the SSE handler.
// It calls ServeSSE and handles any errors by writing an HTTP error response.
func (h *Handler[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.ServeSSE(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler[T]) register() chan []T {
	clientEvtsCh := make(chan []T)
	h.clientRegisterCh <- clientEvtsCh
	return clientEvtsCh
}

func (h *Handler[T]) unregister(clientEvtsCh chan []T) {
	h.clientUnregisterCh <- clientEvtsCh
}

func (h *Handler[T]) replayedEvents() iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		chunk := make([]T, 0, h.cfg.Replay.Initial)
		for evt := range h.evtsReplay.All() {
			chunk = append(chunk, evt)
			if len(chunk) == h.cfg.Replay.Initial {
				if !yield(chunk) {
					return
				}

				// Reset the underlying array, so the length is 0
				chunk = chunk[:0]
			}
		}
		if len(chunk) > 0 {
			yield(chunk)
		}
	}
}

// Broadcast sends an event to all connected clients.
// It returns an error if the handler is closed.
func (h *Handler[T]) Broadcast(evt T) error {
	if h.isClosing() {
		return errors.New("sse-handler: handler is closed")
	}

	h.evtsCh <- evt
	return nil
}

func defaultEventsEncoder[T any](evts []T) ([]byte, error) {
	b, err := json.Marshal(evts)
	if err != nil {
		return nil, fmt.Errorf("sse-handler: unable to encode events: %w", err)
	}
	return b, nil
}
