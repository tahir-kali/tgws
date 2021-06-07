package pool

import (
	"fmt"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	pool := NewPool(16)
	for i := 0; i < 26; i++ {
		j := i //must be like this. otherwise it will all be 26
		pool.Schedule(func() {
			fmt.Println(j)
		})
	}
	time.Sleep(time.Second)
}
