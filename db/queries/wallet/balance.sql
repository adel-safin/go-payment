-- name: GetBalance :one
SELECT wallet_id, balance_minor, version FROM account_balances WHERE wallet_id = $1;

-- name: UpdateBalanceOptimistic :execresult
UPDATE account_balances
SET balance_minor = $1, version = version + 1
WHERE wallet_id = $2 AND version = $3;
