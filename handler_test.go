package sse_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/softwarespot/sse"
)

func Test_Handler(t *testing.T) {
	cfg := sse.NewConfig[string]()
	cfg.FlushFrequency = 128 * time.Millisecond

	h := sse.New(cfg)

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/events", nil)
	go h.ServeHTTP(w1, r1)

	// Should send events to the connected client
	var (
		event1 = "Event 1"
		event2 = "Event 2"
		event3 = "Event 3"
	)
	assertBroadcastEvents(t, h, event1)
	assertBroadcastEvents(t, h, event2, event3)

	res1 := w1.Result()
	defer res1.Body.Close()

	assertEqual(t, res1.StatusCode, http.StatusOK)

	b1, err := io.ReadAll(res1.Body)
	assertNoError(t, err)

	wantRes1 := `data: ["Event 1"]` + "\n\n" + `data: ["Event 2","Event 3"]` + "\n\n"
	assertEqual(t, string(b1), wantRes1)

	// Should replay the events for a connected client

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/events", nil)
	go h.ServeHTTP(w2, r2)

	// Wait for the event(s) to be flushed
	time.Sleep(256 * time.Millisecond)

	res2 := w2.Result()
	defer res2.Body.Close()

	assertEqual(t, res2.StatusCode, http.StatusOK)
	b2, err := io.ReadAll(res2.Body)
	assertNoError(t, err)

	wantRes2 := `data: ["Event 1","Event 2","Event 3"]` + "\n\n"
	assertEqual(t, string(b2), wantRes2)

	assertNoError(t, h.Close())

	// Should return error when the handler is closed
	assertError(t, h.Broadcast(event3))
	assertError(t, h.Close())
}

func assertBroadcastEvents[T any](t testing.TB, h *sse.Handler[T], evts ...T) {
	t.Helper()
	assertNoError(t, h.Broadcast(evts...))

	// Wait for the event(s) to be flushed
	time.Sleep(256 * time.Millisecond)
}
