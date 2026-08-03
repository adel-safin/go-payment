package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestApplyCreditDebit(t *testing.T) {
	b := Balance{BalanceMinor: 1000, Version: 1}
	b, err := ApplyCredit(b, 500)
	if err != nil || b.BalanceMinor != 1500 || b.Version != 2 {
		t.Fatalf("credit: %+v %v", b, err)
	}
	b, err = ApplyDebit(b, 400)
	if err != nil || b.BalanceMinor != 1100 || b.Version != 3 {
		t.Fatalf("debit: %+v %v", b, err)
	}
	if _, err := ApplyDebit(b, 99999); err != ErrInsufficientFunds {
		t.Fatalf("expected insufficient funds, got %v", err)
	}
	if _, err := ApplyCredit(b, 0); err != ErrInvalidAmount {
		t.Fatalf("expected invalid amount")
	}
}

func TestNewWallet(t *testing.T) {
	uid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	w, err := NewWallet(uid, "RUB")
	if err != nil {
		t.Fatal(err)
	}
	if w.Currency != "RUB" {
		t.Fatal(w.Currency)
	}
	if _, err := NewWallet(uid, "RU"); err != ErrInvalidCurrency {
		t.Fatalf("got %v", err)
	}
}
