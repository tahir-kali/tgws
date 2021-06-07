package websocket

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/tahir-kali/gows/pool"
)

func serverHandler(done chan struct{}) {
	channels := NewChannelPool()
	echo := func(c *Channel, op OpCode, data []byte) {
		// echo
		c.Send(op, data)
	}
	wh, _ := NewHandler(echo, channels)
	pool := pool.NewPool(8)
	wh.SetPool(pool)
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// ch, err :=
		wh.Upgrade(w, r)
	})
	svc := &http.Server{
		Handler: mux,
		Addr:    "127.0.0.1:8080",
	}
	go svc.ListenAndServe()
	<-done
	svc.Close()
}

func TestServerHandler(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	go serverHandler(done)
	handler := func(c *Channel, op OpCode, data []byte) {
		// echo
		//c.Send(op, data)
		fmt.Println("data:", string(data))
	}
	onclose := func() {
		fmt.Println("closed")
	}

	wh, _ := NewHandler(handler)
	addr := "ws://127.0.0.1:8080/ws"
	time.Sleep(time.Second)
	ch, _ := wh.Dial(addr)
	ch.SetOnClose(onclose)
	msg := "Hello World"
	err := ch.Send(OpText, []byte(msg))
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Second)
	ch.Close()
	fmt.Println(ch.IsClose())
	time.Sleep(time.Second)
}
