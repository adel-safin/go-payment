package domain

import "testing"

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

func TestBalancedPair(t *testing.T) {
	if !BalancedPair(100, 100) {
		t.Fatal("expected balanced")
	}
	if BalancedPair(100, 99) {
		t.Fatal("expected unbalanced")
	}
}
