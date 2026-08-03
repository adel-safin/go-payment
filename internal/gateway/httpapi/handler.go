package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	identityv1 "github.com/adel-safin/go-payment/api/gen/identity/v1"
	transferv1 "github.com/adel-safin/go-payment/api/gen/transfer/v1"
	walletv1 "github.com/adel-safin/go-payment/api/gen/wallet/v1"
	pkgauth "github.com/adel-safin/go-payment/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Handler struct {
	identity identityv1.IdentityServiceClient
	wallet   walletv1.WalletServiceClient
	transfer transferv1.TransferServiceClient
	tokens   *pkgauth.TokenManager
}

func New(
	identity identityv1.IdentityServiceClient,
	wallet walletv1.WalletServiceClient,
	transfer transferv1.TransferServiceClient,
	tokens *pkgauth.TokenManager,
) http.Handler {
	h := &Handler{identity: identity, wallet: wallet, transfer: transfer, tokens: tokens}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/register", h.register)
		r.Post("/auth/login", h.login)

		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware)
			r.Post("/wallets", h.createWallet)
			r.Get("/wallets/{walletID}/balance", h.getBalance)
			r.Post("/transfers", h.createTransfer)
			r.Get("/transfers/{transferID}", h.getTransfer)
		})
	})

	return otelhttp.NewHandler(r, "gateway")
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	res, err := h.identity.Register(r.Context(), &identityv1.RegisterRequest{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id": res.UserId, "email": res.Email, "role": res.Role,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	res, err := h.identity.Login(r.Context(), &identityv1.LoginRequest{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": res.Token, "user_id": res.UserId, "email": res.Email,
		"role": res.Role, "expires_at": res.ExpiresAtUnix,
	})
}

func (h *Handler) createWallet(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Currency string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Currency == "" {
		body.Currency = "RUB"
	}
	if h.wallet == nil {
		writeErr(w, http.StatusServiceUnavailable, "wallet unavailable")
		return
	}
	res, err := h.wallet.CreateWallet(r.Context(), &walletv1.CreateWalletRequest{
		UserId: claims.UserID, Currency: body.Currency,
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"wallet_id": res.WalletId, "user_id": res.UserId, "currency": res.Currency,
	})
}

func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	if h.wallet == nil {
		writeErr(w, http.StatusServiceUnavailable, "wallet unavailable")
		return
	}
	res, err := h.wallet.GetBalance(r.Context(), &walletv1.GetBalanceRequest{
		WalletId: chi.URLParam(r, "walletID"),
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"wallet_id": res.WalletId, "balance_minor": res.BalanceMinor,
		"currency": res.Currency, "version": res.Version,
	})
}

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		FromWalletID string `json:"from_wallet_id"`
		ToWalletID   string `json:"to_wallet_id"`
		AmountMinor  int64  `json:"amount_minor"`
		Currency     string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if h.transfer == nil {
		writeErr(w, http.StatusServiceUnavailable, "transfer unavailable")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	res, err := h.transfer.CreateTransfer(r.Context(), &transferv1.CreateTransferRequest{
		FromWalletId:   body.FromWalletID,
		ToWalletId:     body.ToWalletID,
		AmountMinor:    body.AmountMinor,
		Currency:       body.Currency,
		IdempotencyKey: key,
		UserId:         claims.UserID,
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transfer_id": res.TransferId, "status": res.Status,
		"amount_minor": res.AmountMinor, "idempotent_replay": res.IdempotentReplay,
	})
}

func (h *Handler) getTransfer(w http.ResponseWriter, r *http.Request) {
	if h.transfer == nil {
		writeErr(w, http.StatusServiceUnavailable, "transfer unavailable")
		return
	}
	res, err := h.transfer.GetTransfer(r.Context(), &transferv1.GetTransferRequest{
		TransferId: chi.URLParam(r, "transferID"),
	})
	if err != nil {
		writeGRPCErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transfer_id": res.TransferId, "from_wallet_id": res.FromWalletId,
		"to_wallet_id": res.ToWalletId, "amount_minor": res.AmountMinor,
		"currency": res.Currency, "status": res.Status,
	})
}

type ctxKey int

const claimsKey ctxKey = 1

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		claims, err := h.tokens.Parse(strings.TrimPrefix(hdr, "Bearer "))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFrom(ctx context.Context) *pkgauth.Claims {
	c, _ := ctx.Value(claimsKey).(*pkgauth.Claims)
	return c
}
