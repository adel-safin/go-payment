package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	tm := NewTokenManager("secret", "go-payment", time.Hour)
	tok, exp, err := tm.Issue("u1", "a@b.c", "user")
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" || exp.IsZero() {
		t.Fatal("expected token and expiry")
	}
	claims, err := tm.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "u1" || claims.Email != "a@b.c" || claims.Role != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestParseInvalid(t *testing.T) {
	tm := NewTokenManager("secret", "go-payment", time.Hour)
	if _, err := tm.Parse("not-a-token"); err == nil {
		t.Fatal("expected error")
	}
}
