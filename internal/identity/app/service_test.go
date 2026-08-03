package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/adel-safin/go-payment/internal/identity/app"
	"github.com/adel-safin/go-payment/internal/identity/domain"
	pkgauth "github.com/adel-safin/go-payment/pkg/auth"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memUsers struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]domain.User
	email map[string]uuid.UUID
}

func newMem() *memUsers {
	return &memUsers{byID: map[uuid.UUID]domain.User{}, email: map[string]uuid.UUID{}}
}

func (m *memUsers) Create(_ context.Context, user domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.email[user.Email]; ok {
		return pkgerrors.ErrAlreadyExists
	}
	m.byID[user.ID] = user
	m.email[user.Email] = user.ID
	return nil
}

func (m *memUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.email[email]
	if !ok {
		return domain.User{}, pkgerrors.ErrNotFound
	}
	return m.byID[id], nil
}

func (m *memUsers) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, pkgerrors.ErrNotFound
	}
	return u, nil
}

func TestGetUser(t *testing.T) {
	svc := app.NewService(newMem(), pkgauth.NewTokenManager("s", "iss", time.Hour))
	reg, err := svc.Register(context.Background(), "b@b.com", "password1")
	require.NoError(t, err)
	u, err := svc.GetUser(context.Background(), reg.UserID)
	require.NoError(t, err)
	require.Equal(t, "b@b.com", u.Email)
	_, err = svc.GetUser(context.Background(), "bad")
	require.Error(t, err)
}
