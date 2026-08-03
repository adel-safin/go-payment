package walletclient

import (
	"context"

	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	"github.com/adel-safin/go-payment/internal/wallet/domain"
	pkgerrors "github.com/adel-safin/go-payment/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	c walletv1.WalletServiceClient
}

func New(c walletv1.WalletServiceClient) *Client {
	return &Client{c: c}
}

func (c *Client) Debit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) error {
	_, err := c.c.Debit(ctx, &walletv1.DebitRequest{
		WalletId: walletID, AmountMinor: amount, TransferId: transferID, IdempotencyKey: idemKey,
	})
	return mapErr(err)
}

func (c *Client) Credit(ctx context.Context, walletID string, amount int64, transferID, idemKey string) error {
	_, err := c.c.Credit(ctx, &walletv1.CreditRequest{
		WalletId: walletID, AmountMinor: amount, TransferId: transferID, IdempotencyKey: idemKey,
	})
	return mapErr(err)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.FailedPrecondition:
		return domain.ErrInsufficientFunds
	case codes.NotFound:
		return pkgerrors.ErrNotFound
	case codes.InvalidArgument:
		return pkgerrors.ErrInvalidArgument
	default:
		return err
	}
}