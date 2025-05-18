package dbx

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

var (
	ErrFailedToBeginTx   = errors.New("failed to begin transaction")
	ErrFailedToCommitTx  = errors.New("failed to commit transaction")
	ErrFailedToExecuteTx = errors.New("failed to execute transaction")
)

type TxFn func(ctx context.Context, tx pgx.Tx) error

func RunInTx(
	ctx context.Context,
	logger *slog.Logger,
	pg *Postgres,
	fn TxFn,
) error {
	tx, err := pg.Pool.Begin(ctx)
	if err != nil {
		return errors.Join(ErrFailedToBeginTx, err)
	}

	defer func() {
		// Rollback is safe to call even if the transaction is already closed/committed
		// If the transaction was already committed, Rollback will return an error
		// which we can safely ignore. Only log if it's a different error.
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logger.ErrorContext(ctx, "failed to rollback transaction", "error", err)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return errors.Join(ErrFailedToExecuteTx, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Join(ErrFailedToCommitTx, err)
	}

	return nil
}
