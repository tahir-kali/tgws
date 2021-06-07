package websocket

import "sync"

type ChannelPool struct {
	m map[string]*Channel
	sync.Mutex
}

func (c *ChannelPool) Set(id string, ch *Channel) {
	c.Lock()
	c.m[id] = ch
	c.Unlock()
}

func (c *ChannelPool) Get(id string) *Channel {
	c.Lock()
	defer c.Unlock()
	return c.m[id]
}

func (c *ChannelPool) Delete(id string) {
	c.Lock()
	delete(c.m, id)
	c.Unlock()
}

func NewChannelPool() *ChannelPool {
	return &ChannelPool{
		m: make(map[string]*Channel),
	}
}
