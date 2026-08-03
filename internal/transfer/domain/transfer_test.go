package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewTransferInvalid(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	if _, err := NewTransfer(a, b, 0, "RUB", "k", uuid.New()); err != ErrInvalidTransfer {
		t.Fatalf("got %v", err)
	}
	if _, err := NewTransfer(a, b, 10, "", "k", uuid.New()); err != ErrInvalidTransfer {
		t.Fatalf("got %v", err)
	}
	if _, err := NewTransfer(a, b, 10, "RUB", "", uuid.New()); err != ErrInvalidTransfer {
		t.Fatalf("got %v", err)
	}
}
