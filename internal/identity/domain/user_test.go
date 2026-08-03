package domain

import "testing"

func TestNewUser(t *testing.T) {
	u, err := NewUser("Alice@Example.com", "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "alice@example.com" {
		t.Fatalf("email: %s", u.Email)
	}
	if u.Role != "user" {
		t.Fatalf("role: %s", u.Role)
	}
	if err := u.CheckPassword("secret123"); err != nil {
		t.Fatal(err)
	}
	if err := u.CheckPassword("wrong"); err == nil {
		t.Fatal("expected bad credentials")
	}
}

func TestNewUserValidation(t *testing.T) {
	if _, err := NewUser("bad", "secret123", ""); err == nil {
		t.Fatal("expected invalid email")
	}
	if _, err := NewUser("a@b.c", "short", ""); err == nil {
		t.Fatal("expected invalid password")
	}
}
