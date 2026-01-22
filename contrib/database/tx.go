package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Tx wraps sql.Tx with additional utilities for transaction management.
// It provides helper methods for commit, rollback, and savepoints.
type Tx struct {
	*sql.Tx
	db *DB
}

// TxOptions contains transaction options for controlling isolation level
// and read-only mode.
//
// Example:
//
//	opts := &database.TxOptions{
//	    Isolation: sql.LevelSerializable,
//	    ReadOnly:  true,
//	}
//	db.WithTxOpts(ctx, opts, func(tx *database.Tx) error {
//	    // Transaction with serializable isolation
//	    return nil
//	})
type TxOptions struct {
	Isolation sql.IsolationLevel // Transaction isolation level
	ReadOnly  bool               // Whether the transaction is read-only
}

// DefaultTxOptions returns default transaction options.
func DefaultTxOptions() *TxOptions {
	return &TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
}

// Begin starts a new transaction.
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	return db.BeginTx(ctx, nil)
}

// BeginTx starts a new transaction with options.
func (db *DB) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {
	var sqlOpts *sql.TxOptions
	if opts != nil {
		sqlOpts = &sql.TxOptions{
			Isolation: opts.Isolation,
			ReadOnly:  opts.ReadOnly,
		}
	}

	tx, err := db.DB.BeginTx(ctx, sqlOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Tx{Tx: tx, db: db}, nil
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction.
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		if err == sql.ErrTxDone {
			return nil // Already committed or rolled back
		}
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// WithTx executes a function within a transaction with automatic commit/rollback.
// If the function returns an error, the transaction is automatically rolled back.
// Otherwise, the transaction is committed. Panics are also caught and trigger a rollback.
//
// This is the recommended way to execute database operations within a transaction.
//
// Example:
//
//	err := db.WithTx(ctx, func(tx *database.Tx) error {
//	    // Insert user
//	    result, err := tx.ExecContext(ctx,
//	        "INSERT INTO users (name, email) VALUES ($1, $2)",
//	        "John Doe", "john@example.com")
//	    if err != nil {
//	        return err  // Automatically rolls back
//	    }
//
//	    userID, _ := result.LastInsertId()
//
//	    // Insert user profile
//	    _, err = tx.ExecContext(ctx,
//	        "INSERT INTO profiles (user_id, bio) VALUES ($1, $2)",
//	        userID, "Software engineer")
//	    return err  // Commits if nil, rolls back if error
//	})
func (db *DB) WithTx(ctx context.Context, fn func(*Tx) error) error {
	return db.WithTxOpts(ctx, nil, fn)
}

// WithTxOpts executes a function within a transaction with custom options.
// Allows specifying isolation level and read-only mode.
//
// Example:
//
//	opts := &database.TxOptions{
//	    Isolation: sql.LevelSerializable,
//	}
//	err := db.WithTxOpts(ctx, opts, func(tx *database.Tx) error {
//	    // Transaction with serializable isolation
//	    return updateInventory(ctx, tx, productID, quantity)
//	})
func (db *DB) WithTxOpts(ctx context.Context, opts *TxOptions, fn func(*Tx) error) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// TxFunc is a function that executes within a transaction.
type TxFunc func(*Tx) error

// Chain chains multiple transaction functions together.
// All functions run in the same transaction.
func Chain(fns ...TxFunc) TxFunc {
	return func(tx *Tx) error {
		for _, fn := range fns {
			if err := fn(tx); err != nil {
				return err
			}
		}
		return nil
	}
}

// Savepoint creates a savepoint within the transaction.
// Note: Not all databases support savepoints.
func (tx *Tx) Savepoint(name string) error {
	_, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", name))
	return err
}

// RollbackTo rolls back to a savepoint.
func (tx *Tx) RollbackTo(name string) error {
	_, err := tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", name))
	return err
}

// ReleaseSavepoint releases a savepoint.
func (tx *Tx) ReleaseSavepoint(name string) error {
	_, err := tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", name))
	return err
}
