package database

import (
	"context"
	"database/sql"
	"fmt"

	"gopherswitch/internal/modules/auth/domain"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) FindByCardID(ctx context.Context, cardID int64) (*domain.Account, error) {
	query := `
		SELECT id, account_number, card_id, balance, avail_balance,
			   currency_code, status, created_at, updated_at
		FROM accounts WHERE card_id = $1`

	var account domain.Account
	err := r.db.QueryRowContext(ctx, query, cardID).Scan(
		&account.ID, &account.AccountNumber, &account.CardID, &account.Balance, &account.AvailBalance,
		&account.CurrencyCode, &account.Status, &account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account for card ID %d not found", cardID)
		}
		return nil, fmt.Errorf("failed to find account by card ID: %w", err)
	}

	return &account, nil
}

func (r *AccountRepository) Debit(ctx context.Context, accountID int64, amount int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current balance
	var currentBalance int64
	err = tx.QueryRowContext(ctx, "SELECT avail_balance FROM accounts WHERE id = $1 FOR UPDATE", accountID).Scan(&currentBalance)
	if err != nil {
		return 0, fmt.Errorf("failed to get current balance: %w", err)
	}

	if currentBalance < amount {
		return 0, fmt.Errorf("insufficient funds")
	}

	// Update balance
	newBalance := currentBalance - amount
	_, err = tx.ExecContext(ctx, 
		"UPDATE accounts SET balance = balance - $1, avail_balance = avail_balance - $1, updated_at = NOW() WHERE id = $2",
		amount, accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to debit account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newBalance, nil
}

func (r *AccountRepository) Credit(ctx context.Context, accountID int64, amount int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current balance
	var currentBalance int64
	err = tx.QueryRowContext(ctx, "SELECT avail_balance FROM accounts WHERE id = $1 FOR UPDATE", accountID).Scan(&currentBalance)
	if err != nil {
		return 0, fmt.Errorf("failed to get current balance: %w", err)
	}

	// Update balance
	newBalance := currentBalance + amount
	_, err = tx.ExecContext(ctx, 
		"UPDATE accounts SET balance = balance + $1, avail_balance = avail_balance + $1, updated_at = NOW() WHERE id = $2",
		amount, accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to credit account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newBalance, nil
}

func (r *AccountRepository) GetBalance(ctx context.Context, accountID int64) (int64, error) {
	query := `SELECT avail_balance FROM accounts WHERE id = $1`

	var balance int64
	err := r.db.QueryRowContext(ctx, query, accountID).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("account with ID %d not found", accountID)
		}
		return 0, fmt.Errorf("failed to get account balance: %w", err)
	}

	return balance, nil
}

func (r *AccountRepository) Create(ctx context.Context, account *domain.Account) error {
	query := `
		INSERT INTO accounts (
			account_number, card_id, balance, avail_balance,
			currency_code, status
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		account.AccountNumber, account.CardID, account.Balance, account.AvailBalance,
		account.CurrencyCode, account.Status,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

func (r *AccountRepository) FindByAccountNumber(ctx context.Context, accountNumber string) (*domain.Account, error) {
	query := `
		SELECT id, account_number, card_id, balance, avail_balance,
			   currency_code, status, created_at, updated_at
		FROM accounts WHERE account_number = $1`

	var account domain.Account
	err := r.db.QueryRowContext(ctx, query, accountNumber).Scan(
		&account.ID, &account.AccountNumber, &account.CardID, &account.Balance, &account.AvailBalance,
		&account.CurrencyCode, &account.Status, &account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account with number %s not found", accountNumber)
		}
		return nil, fmt.Errorf("failed to find account by number: %w", err)
	}

	return &account, nil
}
