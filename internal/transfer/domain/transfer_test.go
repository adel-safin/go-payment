package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewTransfer(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	tr, err := NewTransfer(a, b, 100, "RUB", "key", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if tr.Status != StatusPending {
		t.Fatalf("status %s", tr.Status)
	}
	if _, err := NewTransfer(a, a, 100, "RUB", "k", uuid.New()); err != ErrSameWallet {
		t.Fatalf("expected same wallet err, got %v", err)
	}
}
