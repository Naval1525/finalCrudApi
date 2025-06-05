// repository.go - Database repository layer
// This layer handles all database operations
package main

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository interface defines the contract for database operations
// Using interfaces makes our code more testable and maintainable
type Repository interface {
	// User operations
	CreateUser(user *User) error
	GetUserByID(id int) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUsers(limit, offset int) ([]*User, error)
	UpdateUser(id int, updates map[string]interface{}) error
	DeleteUser(id int) error
	GetUserCount() (int, error)
}

// repository implements the Repository interface
type repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// CreateUser creates a new user in the database
func (r *repository) CreateUser(user *User) error {
	// SQL query to insert a new user
	query := `
		INSERT INTO users (username, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	// Execute the query and scan the returned values
	err := r.db.QueryRow(
		query,
		user.Username,
		user.Email,
		user.Password,
		time.Now(),
		time.Now(),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID
func (r *repository) GetUserByID(id int) (*User, error) {
	user := &User{}

	query := `
		SELECT id, username, email, password, created_at, updated_at
		FROM users
		WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *repository) GetUserByEmail(email string) (*User, error) {
	user := &User{}

	query := `
		SELECT id, username, email, password, created_at, updated_at
		FROM users
		WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUsers retrieves a list of users with pagination
func (r *repository) GetUsers(limit, offset int) ([]*User, error) {
	query := `
		SELECT id, username, email, password, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	// Execute query
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close() // Always close rows when done

	var users []*User

	// Iterate through rows
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// UpdateUser updates a user's information
func (r *repository) UpdateUser(id int, updates map[string]interface{}) error {
	// Build dynamic query based on which fields are being updated
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	// Add each update to the query
	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Add updated_at timestamp
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add user ID for WHERE clause
	args = append(args, id)

	// Build final query
	query := fmt.Sprintf(`
		UPDATE users
		SET %s
		WHERE id = $%d`,
		joinStrings(setParts, ", "),
		argIndex,
	)

	// Execute update
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// DeleteUser deletes a user from the database
func (r *repository) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetUserCount returns the total number of users
func (r *repository) GetUserCount() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`

	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}

	return count, nil
}

// Helper function to join strings (like strings.Join but inline)
func joinStrings(strings []string, separator string) string {
	if len(strings) == 0 {
		return ""
	}
	if len(strings) == 1 {
		return strings[0]
	}

	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += separator + strings[i]
	}
	return result
}
