package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
	ErrVersionConflict     = errors.New("version conflict")
	ErrAlreadyExists       = errors.New("already exists")
)

func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}
