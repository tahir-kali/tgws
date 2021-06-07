package websocket

import (
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"sync"
	"unsafe"

	"github.com/tahir-kali/tgws"
 	"https://github.com/tahir-kali/tgws/wsutil/"
	"github.com/mailru/easygo/netpoll"
)

// Channel is a websocket connection that messages are read from and written to.
type Channel struct {
	handler *Handler // Handle managing this connection.
	conn    net.Conn // Websocket connection.

	readDesc *netpoll.Desc // Read descriptor for netpoll.

	// onClose is called when the channel is closed.
	onClose    func()
	onCloseMux sync.Mutex
	isClient   bool
	isClose    bool
}

// getConnFromTLSConn returns the internal wrapped connection from the tls.Conn.
func getConnFromTLSConn(tlsConn *tls.Conn) net.Conn {
	// XXX: This is really BAD!!! Only way currently to get the underlying
	// connection of the tls.Conn. At least until
	// https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(net.Conn)
}

// Create a new channel for the connection.
func newChannel(conn net.Conn, handler *Handler, isClient bool) *Channel {
	fdConn := conn
	tlsConn, ok := conn.(*tls.Conn)
	if ok {
		fdConn = getConnFromTLSConn(tlsConn)
	}
	return &Channel{
		handler:  handler,
		conn:     conn,
		readDesc: netpoll.Must(netpoll.HandleReadOnce(fdConn)),
		isClient: isClient,
	}
}

func (c *Channel) IsClose() bool {
	return c.isClose
}

// Close the channel.
func (c *Channel) Close() {
	c.handler.poller.Stop(c.readDesc)
	c.conn.Close()
	c.isClose = true
	c.onCloseMux.Lock()
	defer c.onCloseMux.Unlock()
	if c.onClose != nil {
		c.onClose()
	}
}

// Send a message over the channel. Once write concurrency of the handler is
// reached this method will block.
func (c *Channel) Send(op OpCode, data []byte) error {
	if c.isClient {
		return wsutil.WriteClientMessage(c.conn, ws.OpCode(op), data)
	}
	return wsutil.WriteServerMessage(c.conn, ws.OpCode(op), data)
}

// SetOnClose sets the callback to get called when the channel is closed.
func (c *Channel) SetOnClose(callback func()) {
	c.onCloseMux.Lock()
	defer c.onCloseMux.Unlock()
	c.onClose = callback
}

// Read and process the message from the connection.
func (c *Channel) read() {
	var data []byte
	var op ws.OpCode
	var err error
	if c.isClient {
		data, op, err = wsutil.ReadServerData(c.conn)
	} else {
		data, op, err = wsutil.ReadClientData(c.conn)
	}
	if err != nil {
		fmt.Println(err)
		c.Close()
		return
	}

	c.handler.callback(c, OpCode(op), data)
}
