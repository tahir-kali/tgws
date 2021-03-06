package pool

// Pool of goroutines that have a max concurrency.
type Pool struct {
	work chan func()
	sem  chan struct{}
}

// Create a new pool.
func NewPool(concurrency int) *Pool {
	return &Pool{
		work: make(chan func()),
		sem:  make(chan struct{}, concurrency),
	}
}

// Schedule a task on the pool.
func (p *Pool) Schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

// Perform the scheduled work.
func (p *Pool) worker(task func()) {
	defer func() { <-p.sem }()
	for {
		task()
		task = <-p.work
	}
}
