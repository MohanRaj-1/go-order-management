package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type TransactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

func (m *TransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(*sql.Tx) error,
) error {

	if fn == nil {
		return errors.New("transaction callback cannot be nil")
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Safety net: ensures unfinished transactions are rolled back,
	// including when the callback panics.
	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil &&
			!errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(
				fmt.Errorf("execute transaction: %w", err),
				fmt.Errorf("rollback transaction: %w", rollbackErr),
			)
		}

		return fmt.Errorf("execute transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
