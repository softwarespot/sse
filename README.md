# Server-Sent Events (SSE) handler

[![Go Reference](https://pkg.go.dev/badge/github.com/softwarespot/sse.svg)](https://pkg.go.dev/github.com/softwarespot/sse) ![Go Tests](https://github.com/softwarespot/replay/actions/workflows/go.yml/badge.svg)

**Server-Sent Events (SSE) handler** is a generic compatible and [http.Handler](https://pkg.go.dev/net/http#Handler) compliant module, that implements real-time event streaming from a server to web clients using the [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) protocol. It's a robust module for managing multiple client connections, broadcasting events, and handling client registrations and unregistrations efficiently.

Examples of using this module can be found from the [./examples](./examples/) directory.

## Prerequisites

- Go 1.25.0 or above

## Installation

```bash
go get -u github.com/softwarespot/sse
```

## Usage

A basic example of using **SSE**.

```Go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/softwarespot/sse"
)

func main() {
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
```

## License

The code has been licensed under the [MIT](https://opensource.org/license/mit) license.
