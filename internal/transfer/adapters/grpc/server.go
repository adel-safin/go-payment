package grpcadapter

import (
	"context"
	"errors"

	transferv1 "github.com/adel-safin/go-payment/api/gen/transfer/v1"
	"github.com/adel-safin/go-payment/internal/transfer/app"
	"github.com/adel-safin/go-payment/internal/transfer/domain"
	walletdomain "github.com/adel-safin/go-payment/internal/wallet/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	transferv1.UnimplementedTransferServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateTransfer(ctx context.Context, req *transferv1.CreateTransferRequest) (*transferv1.CreateTransferResponse, error) {
	res, err := s.svc.Create(ctx, req.GetFromWalletId(), req.GetToWalletId(), req.GetAmountMinor(), req.GetCurrency(), req.GetIdempotencyKey(), req.GetUserId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &transferv1.CreateTransferResponse{
		TransferId:       res.Transfer.ID.String(),
		Status:           string(res.Transfer.Status),
		AmountMinor:      res.Transfer.AmountMinor,
		IdempotentReplay: res.IdempotentReplay,
	}, nil
}

func (s *Server) GetTransfer(ctx context.Context, req *transferv1.GetTransferRequest) (*transferv1.GetTransferResponse, error) {
	tr, err := s.svc.Get(ctx, req.GetTransferId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &transferv1.GetTransferResponse{
		TransferId:   tr.ID.String(),
		FromWalletId: tr.FromWalletID.String(),
		ToWalletId:   tr.ToWalletID.String(),
		AmountMinor:  tr.AmountMinor,
		Currency:     tr.Currency,
		Status:       string(tr.Status),
	}, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, walletdomain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrSameWallet), errors.Is(err, domain.ErrInvalidTransfer), errors.Is(err, pkgerrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pkgerrors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pkgerrors.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error")
	}
}
