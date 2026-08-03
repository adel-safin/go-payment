package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/adel-safin/go-payment/internal/identity/domain"
	"github.com/adel-safin/go-payment/internal/identity/ports"
	pkgauth "github.com/adel-safin/go-payment/pkg/auth"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"github.com/google/uuid"
)

type Service struct {
	users  ports.UserRepository
	tokens *pkgauth.TokenManager
}

func NewService(users ports.UserRepository, tokens *pkgauth.TokenManager) *Service {
	return &Service{users: users, tokens: tokens}
}

type RegisterResult struct {
	UserID string
	Email  string
	Role   string
}

type LoginResult struct {
	Token     string
	UserID    string
	Email     string
	Role      string
	ExpiresAt time.Time
}

func (s *Service) Register(ctx context.Context, email, password string) (RegisterResult, error) {
	user, err := domain.NewUser(email, password, "user")
	if err != nil {
		return RegisterResult{}, pkgerrors.Wrap(pkgerrors.ErrInvalidArgument, err.Error())
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, pkgerrors.ErrAlreadyExists) {
			return RegisterResult{}, err
		}
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: user.ID.String(), Email: user.Email, Role: user.Role}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			return LoginResult{}, domain.ErrBadCredentials
		}
		return LoginResult{}, err
	}
	if err := user.CheckPassword(password); err != nil {
		return LoginResult{}, err
	}
	token, exp, err := s.tokens.Issue(user.ID.String(), user.Email, user.Role)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token:     token,
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: exp,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (domain.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return domain.User{}, pkgerrors.ErrInvalidArgument
	}
	return s.users.GetByID(ctx, id)
}
