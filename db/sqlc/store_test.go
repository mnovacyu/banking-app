package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	// Run n concurrent transfer transactions
	n := 5
	amount := int64(10)

	// Use channels to connect concurrent Go routines, allows them to safely share data with each other wihtout explicit locking. in our case, we need one channel to receive the errors, and one channel to receive the TransferTxResult. Use make keyword to create the channel
	errs := make(chan error)
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {

		go func() {
			// Cannot use testify require to check result or error, because function is running inside a different Go routine as the one TestTransferTx function is running on. So no guarantee that it will stop the whole test if a condition is not satisfied. Correct way to verify error and result is to send them back to the main go routine that our test is running on. We use channels for that.
			result, err := store.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:		   amount,
			})

			// Send err to the errors channel, and result to the results channel
			errs <- err
			results <- result
		}()
	}

	// check results
	for i := 0; i < n; i++ {
		// This time the channel is on the right of the arrow. Variable is on the left to receive the stored data
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// Check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// Check from entry
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		// Check to entry
		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// TODO: check accounts' balance
	}
}