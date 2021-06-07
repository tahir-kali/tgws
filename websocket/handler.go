package websocket

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/tahir-kali/tgws"
	"github.com/mailru/easygo/netpoll"
	"github.com/tahir-kali/tgws/pool"
)

// Handler is a high performance websocket handler that uses netpoll with
// read and write concurrency to allow a high number of concurrent websocket
// connections.
type Handler struct {
	callback RecievedCallback
	poller   netpoll.Poller
	pool     *pool.Pool
	channels *ChannelPool
}

// OpCode represents an operation code.
type OpCode ws.OpCode

// Operation codes that will be used in the RecievedCallback. rfc6455 defines
// more operation codes but those are hidden by the implementation.
const (
	OpText   OpCode = 0x1
	OpBinary OpCode = 0x2
)

// RecievedCallback is the signature for the callback called when a message
// is recieved on a Channel.
type RecievedCallback func(c *Channel, op OpCode, data []byte)

// NewHandler creates a new websocket handler.
func NewHandler(callback RecievedCallback, channels *ChannelPool) (*Handler, error) {
	// Creates the poller that is used when a websocket connects. This is used
	// to prevent the spawning of a 2 goroutines per connection.
	poller, err := netpoll.New(nil)
	if err != nil {
		return nil, err
	}

	// Create the handle object.
	return &Handler{
		callback: callback,
		poller:   poller,
		channels: channels,
	}, nil
}

func (h *Handler) SetPool(pool *pool.Pool) {
	h.pool = pool
}

// CreateChannel upgrades the incoming http request into a websocket channel.
func (h *Handler) Dial(addr string) (*Channel, error) {
	conn, _, _, err := ws.Dial(context.Background(), addr)
	if err != nil {
		return nil, err
	}

	// Open the channel with the client.
	ch := newChannel(conn, h, true)
	h.startRead(ch)
	return ch, nil
}

func (h *Handler) Upgrade(w http.ResponseWriter, r *http.Request) (*Channel, error) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// Ignore the error, the UpdateHTTP handled notifying the client.
		return nil, err
	}
	if len(r.URL.String()) < 1 {
		w.WriteHeader(http.StatusForbidden)
	}

	connID := strings.TrimPrefix(r.URL.Path[1:], "ws/")
	fmt.Println("connID: ", connID)
	// get existing channel or Open new channel with the client.
	ch := h.channels.Get(connID)
	if ch == nil || ch.IsClose() {
		ch = newChannel(conn, h, false)
		h.channels.Set(connID, ch)
	}
	h.startRead(ch)
	return ch, nil
}

// Start the reading of the connection from epoll.
func (h *Handler) startRead(c *Channel) error {
	err := h.poller.Start(c.readDesc, func(ev netpoll.Event) {
		// Verify the connection is not closed.
		if ev&netpoll.EventReadHup != 0 {
			// Connection closed.
			c.Close()
			return
		}

		// Trigger the read. This will block once read concurrency is hit.

		// Actually read from the channel.
		onRecv := func() {
			c.read()

			// Resume to get the next message.
			err := h.poller.Resume(c.readDesc)
			if err != nil {
				// Failed to resume reading, close the connection.
				c.Close()
			}
		}
		if h.pool != nil {
			h.pool.Schedule(onRecv)
		} else {
			onRecv()
		}

	})
	if err == netpoll.ErrRegistered {
		// Already being handled.
	} else if err == netpoll.ErrClosed {
		// Already closed.
	} else if err != nil {
		// Failed to start reading.
		c.Close()
	}
	return err
}
