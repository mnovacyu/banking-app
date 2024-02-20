package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all function to execute db queries and transactions. Golang favors composition over inheritance.
// Golang does not have classes. Instead uses structs to define objects and interfaces to define behavior. Composition is achieved by embedding one struct into another.
type Store struct {
	*Queries
	db *sql.DB
}

// NewStore creates a new Store
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction. Takes a context and a callback function as input.
// execTx starts with a lower e because it's an unexported function, meaning we don't want external packages to call it directly. Instead will provide an exported function for each specific transaction.
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	// Create a new Queries object with the transaction
	q := New(tx)
	// Call the callback function with the created Queries
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount 		  int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer	Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount	Account	 `json:"to_account"`
	FromEntry	Entry    `json:"from_entry"`
	ToEntry		Entry	 `json:"to_entry"`
}

// TransferTx performs a money transfer from one account to the other.
// It creates a transfer record, add account entries, and updates the accounts' balance within a single database transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// Accessing the result variable of the outer function from inside this callback function. Same for argument variable. Makes the callback function become a closure. Since Go lacks support for generics type, closure is often used when we want to get the result from a callback function, because the callback function itself doesn't know the exact type of the result it should return.
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:		   arg.Amount,
		})
		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// TODO: update accounts' balance

		return nil
	})

	return result, err
}