package grpcadapter

import (
	"context"
	"errors"

	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	"github.com/adel-safin/go-payment/internal/wallet/app"
	"github.com/adel-safin/go-payment/internal/wallet/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	walletv1.UnimplementedWalletServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateWallet(ctx context.Context, req *walletv1.CreateWalletRequest) (*walletv1.CreateWalletResponse, error) {
	w, err := s.svc.CreateWallet(ctx, req.GetUserId(), req.GetCurrency())
	if err != nil {
		return nil, mapErr(err)
	}
	return &walletv1.CreateWalletResponse{
		WalletId: w.ID.String(), UserId: w.UserID.String(), Currency: w.Currency,
	}, nil
}

func (s *Server) GetBalance(ctx context.Context, req *walletv1.GetBalanceRequest) (*walletv1.GetBalanceResponse, error) {
	b, w, err := s.svc.GetBalance(ctx, req.GetWalletId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &walletv1.GetBalanceResponse{
		WalletId: b.WalletID.String(), BalanceMinor: b.BalanceMinor,
		Currency: w.Currency, Version: b.Version,
	}, nil
}

func (s *Server) Credit(ctx context.Context, req *walletv1.CreditRequest) (*walletv1.CreditResponse, error) {
	res, err := s.svc.Credit(ctx, req.GetWalletId(), req.GetAmountMinor(), req.GetTransferId(), req.GetIdempotencyKey())
	if err != nil {
		return nil, mapErr(err)
	}
	return &walletv1.CreditResponse{
		WalletId: res.Balance.WalletID.String(), BalanceMinor: res.Balance.BalanceMinor,
		Version: res.Balance.Version, EntryId: res.EntryID.String(),
	}, nil
}

func (s *Server) Debit(ctx context.Context, req *walletv1.DebitRequest) (*walletv1.DebitResponse, error) {
	res, err := s.svc.Debit(ctx, req.GetWalletId(), req.GetAmountMinor(), req.GetTransferId(), req.GetIdempotencyKey())
	if err != nil {
		return nil, mapErr(err)
	}
	return &walletv1.DebitResponse{
		WalletId: res.Balance.WalletID.String(), BalanceMinor: res.Balance.BalanceMinor,
		Version: res.Balance.Version, EntryId: res.EntryID.String(),
	}, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrInvalidCurrency), errors.Is(err, pkgerrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pkgerrors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pkgerrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrVersionConflict):
		return status.Error(codes.Aborted, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
