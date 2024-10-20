package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/softwarespot/sse"
)

func main() {
	// Use the default configuration
	h := sse.New[int64](nil)
	defer h.Close()

	go func() {
		for {
			evt1 := rand.Int64N(1000)
			evt2 := rand.Int64N(1000)
			evt3 := rand.Int64N(1000)
			fmt.Println("sse handler: broadcast event", h.Broadcast(evt1, evt2, evt3))
			time.Sleep(64 * time.Millisecond)
		}
	}()

	http.Handle("/events", h)
	http.ListenAndServe(":3000", nil)
}
