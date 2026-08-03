package breaker

import (
	"testing"
	"time"
)

func TestCircuitOpens(t *testing.T) {
	c := New(2, 50*time.Millisecond)
	if !c.Allow() {
		t.Fatal("should allow")
	}
	c.Failure()
	c.Failure()
	if c.Allow() {
		t.Fatal("should be open")
	}
	time.Sleep(60 * time.Millisecond)
	if !c.Allow() {
		t.Fatal("should half-open/allow after cooldown")
	}
	c.Success()
	if !c.Allow() {
		t.Fatal("should allow after success")
	}
}
