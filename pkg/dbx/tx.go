package dbx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type TxFn func(ctx context.Context, tx pgx.Tx) error

func RunInTx(
	ctx context.Context,
	pg *Postgres,
	fn TxFn,
) error {
	tx, err := pg.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres - RunInTx - pg.Pool.Begin: %w", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			fmt.Println("postgres - RunInTx - tx.Rollback: ", err)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("postgres - RunInTx - fn: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres - RunInTx - tx.Commit: %w", err)
	}

	return nil
}
