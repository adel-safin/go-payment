package grpcadapter

import (
	"context"
	"errors"

	identityv1 "github.com/adel-safin/go-payment/api/gen/identity/v1"
	"github.com/adel-safin/go-payment/internal/identity/app"
	"github.com/adel-safin/go-payment/internal/identity/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(ctx context.Context, req *identityv1.RegisterRequest) (*identityv1.RegisterResponse, error) {
	res, err := s.svc.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapErr(err)
	}
	return &identityv1.RegisterResponse{UserId: res.UserID, Email: res.Email, Role: res.Role}, nil
}

func (s *Server) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	res, err := s.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapErr(err)
	}
	return &identityv1.LoginResponse{
		Token:          res.Token,
		UserId:         res.UserID,
		Email:          res.Email,
		Role:           res.Role,
		ExpiresAtUnix:  res.ExpiresAt.Unix(),
	}, nil
}

func (s *Server) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	u, err := s.svc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &identityv1.GetUserResponse{UserId: u.ID.String(), Email: u.Email, Role: u.Role}, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrBadCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, pkgerrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, pkgerrors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pkgerrors.ErrInvalidArgument),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error")
	}
}
