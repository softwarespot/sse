package sse_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/softwarespot/sse"
)

func ExampleNew() {
	// Use the default configuration
	h := sse.New[int](nil)
	defer h.Close()

	go func() {
		var evt int
		for {
			fmt.Println("sse handler: broadcast event", h.Broadcast(evt))
			evt++
			time.Sleep(64 * time.Millisecond)
		}
	}()

	http.Handle("/events", h)
	http.ListenAndServe(":3000", nil)
}
