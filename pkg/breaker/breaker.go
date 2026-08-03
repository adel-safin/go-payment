package breaker

import (
	"sync"
	"time"
)

// Circuit is a simple consecutive-failure circuit breaker.
type Circuit struct {
	mu           sync.Mutex
	failures     int
	threshold    int
	openUntil    time.Time
	cooldown     time.Duration
}

func New(threshold int, cooldown time.Duration) *Circuit {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 5 * time.Second
	}
	return &Circuit{threshold: threshold, cooldown: cooldown}
}

func (c *Circuit) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.openUntil) {
		return false
	}
	return true
}

func (c *Circuit) Success() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openUntil = time.Time{}
}

func (c *Circuit) Failure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.failures >= c.threshold {
		c.openUntil = time.Now().Add(c.cooldown)
		c.failures = 0
	}
}
