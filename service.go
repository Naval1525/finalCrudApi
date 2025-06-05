// service.go - Business logic layer
// This layer handles business rules, validation, and coordinates between handlers and repository
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Service interface defines the contract for business logic operations
type Service interface {
	// Authentication operations
	Register(req *RegisterRequest) (*User, error)
	Login(req *LoginRequest) (string, error) // returns JWT token

	// User operations
	GetUser(id int) (*User, error)
	GetUsers(page, limit int) (*PaginatedUsers, error)
	UpdateUser(id int, req *UpdateUserRequest) (*User, error)
	DeleteUser(id int) error

	// Background operations (using goroutines)
	ProcessUserAnalytics(userID int)
	GetUserStatistics() (*UserStatistics, error)
}

// service implements the Service interface
type service struct {
	repo      Repository
	jwtSecret string

	// For goroutine examples - tracking background operations
	analyticsQueue chan int
	wg             sync.WaitGroup
}

// PaginatedUsers represents paginated user results
type PaginatedUsers struct {
	Users      []*User `json:"users"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int     `json:"total"`
	TotalPages int     `json:"total_pages"`
}

// UserStatistics represents user analytics data
type UserStatistics struct {
	TotalUsers     int `json:"total_users"`
	RecentUsers    int `json:"recent_users"` // Users created in last 7 days
	ProcessedToday int `json:"processed_today"`
	BackgroundJobs int `json:"background_jobs"`
}

// NewService creates a new service instance
func NewService(repo Repository) Service {
	config := LoadConfig()

	s := &service{
		repo:           repo,
		jwtSecret:      config.JWTSecret,
		analyticsQueue: make(chan int, 100), // Buffered channel for background processing
	}

	// Start background worker goroutines
	s.startBackgroundWorkers()

	return s
}

// startBackgroundWorkers starts goroutines for background processing
func (s *service) startBackgroundWorkers() {
	// Start 3 worker goroutines to process analytics
	for i := 0; i < 3; i++ {
		go s.analyticsWorker(i)
	}
}

// analyticsWorker is a goroutine that processes user analytics in the background
func (s *service) analyticsWorker(workerID int) {
	for userID := range s.analyticsQueue {
		// Simulate some analytics processing
		// In a real app, this might update user stats, send emails, etc.
		fmt.Printf("Worker %d processing analytics for user %d\n", workerID, userID)

		// Simulate some work
		time.Sleep(100 * time.Millisecond)

		// In a real application, you might:
		// - Update user statistics in the database
		// - Send welcome emails
		// - Process user behavior data
		// - Generate reports

		fmt.Printf("Worker %d completed analytics for user %d\n", workerID, userID)
	}
}

// Register creates a new user account
func (s *service) Register(req *RegisterRequest) (*User, error) {
	// Check if user already exists
	existingUser, _ := s.repo.GetUserByEmail(req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Hash the password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user object
	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	// Save user to database
	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Process user analytics in background (using goroutine)
	s.ProcessUserAnalytics(user.ID)

	// Don't return password in response
	user.Password = ""

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *service) Login(req *LoginRequest) (string, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	// Compare password
	if err := ComparePassword(user.Password, req.Password); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token, err := GenerateJWT(user.ID, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Process login analytics in background
	go func() {
		// This is a simple goroutine example
		// In a real app, you might log the login, update last_login timestamp, etc.
		fmt.Printf("User %d logged in at %s\n", user.ID, time.Now().Format(time.RFC3339))

		// You could also add this to a more sophisticated queue
		select {
		case s.analyticsQueue <- user.ID:
			// Successfully queued for processing
		default:
			// Queue is full, handle gracefully
			fmt.Printf("Analytics queue full, skipping for user %d\n", user.ID)
		}
	}()

	return token, nil
}

// GetUser retrieves a user by ID
func (s *service) GetUser(id int) (*User, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// Don't return password
	user.Password = ""

	return user, nil
}

// GetUsers retrieves a paginated list of users
func (s *service) GetUsers(page, limit int) (*PaginatedUsers, error) {
	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Use goroutines to fetch users and count concurrently for better performance
	var users []*User
	var total int
	var userErr, countErr error

	// WaitGroup to wait for both goroutines to complete
	var wg sync.WaitGroup
	wg.Add(2)

	// Fetch users in a goroutine
	go func() {
		defer wg.Done()
		users, userErr = s.repo.GetUsers(limit, offset)

		// Remove passwords from all users
		for _, user := range users {
			user.Password = ""
		}
	}()

	// Get total count in another goroutine
	go func() {
		defer wg.Done()
		total, countErr = s.repo.GetUserCount()
	}()

	// Wait for both operations to complete
	wg.Wait()

	// Check for errors
	if userErr != nil {
		return nil, fmt.Errorf("failed to get users: %w", userErr)
	}
	if countErr != nil {
		return nil, fmt.Errorf("failed to get user count: %w", countErr)
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	return &PaginatedUsers{
		Users:      users,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// UpdateUser updates a user's information
func (s *service) UpdateUser(id int, req *UpdateUserRequest) (*User, error) {
	// Check if user exists
	existingUser, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// Check if email is being changed and if it's already taken
	if req.Email != "" && req.Email != existingUser.Email {
		if existingUser, _ := s.repo.GetUserByEmail(req.Email); existingUser != nil {
			return nil, fmt.Errorf("email %s is already taken", req.Email)
		}
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	// If no updates provided, return error
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	// Perform update
	if err := s.repo.UpdateUser(id, updates); err != nil {
		return nil, err
	}

	// Get updated user
	updatedUser, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// Process update analytics in background
	go func() {
		fmt.Printf("User %d profile updated at %s\n", id, time.Now().Format(time.RFC3339))
		// You could track what fields were updated, send notifications, etc.
	}()

	// Don't return password
	updatedUser.Password = ""

	return updatedUser, nil
}

// DeleteUser deletes a user account
func (s *service) DeleteUser(id int) error {
	// Check if user exists
	_, err := s.repo.GetUserByID(id)
	if err != nil {
		return err
	}

	// Delete user
	if err := s.repo.DeleteUser(id); err != nil {
		return err
	}

	// Process deletion analytics in background
	go func() {
		fmt.Printf("User %d deleted at %s\n", id, time.Now().Format(time.RFC3339))
		// In a real app, you might:
		// - Clean up user data
		// - Send confirmation emails
		// - Update analytics
		// - Log the deletion for audit purposes
	}()

	return nil
}

// ProcessUserAnalytics queues user analytics processing
func (s *service) ProcessUserAnalytics(userID int) {
	// This is non-blocking - if queue is full, we skip
	select {
	case s.analyticsQueue <- userID:
		fmt.Printf("Queued analytics processing for user %d\n", userID)
	default:
		fmt.Printf("Analytics queue full, skipping user %d\n", userID)
	}
}

// GetUserStatistics returns user statistics (demonstrates concurrent processing)
func (s *service) GetUserStatistics() (*UserStatistics, error) {
	stats := &UserStatistics{}

	// Use context with timeout for all operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use goroutines to fetch different statistics concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect concurrent writes to stats
	var errors []error

	// Get total users
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Simulate work with context
		select {
		case <-ctx.Done():
			mu.Lock()
			errors = append(errors, fmt.Errorf("timeout getting total users"))
			mu.Unlock()
			return
		default:
		}

		total, err := s.repo.GetUserCount()
		mu.Lock()
		if err != nil {
			errors = append(errors, err)
		} else {
			stats.TotalUsers = total
		}
		mu.Unlock()
	}()

	// Get recent users (simulated - in real app you'd have a date filter)
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			mu.Lock()
			errors = append(errors, fmt.Errorf("timeout getting recent users"))
			mu.Unlock()
			return
		default:
		}

		// Simulate getting recent users count
		// In a real app, you'd modify your repository to filter by date
		time.Sleep(50 * time.Millisecond) // Simulate DB query

		mu.Lock()
		stats.RecentUsers = 5 // Simulated value
		mu.Unlock()
	}()

	// Get background job stats
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			mu.Lock()
			errors = append(errors, fmt.Errorf("timeout getting background job stats"))
			mu.Unlock()
			return
		default:
		}

		mu.Lock()
		stats.ProcessedToday = 42 // Simulated value
		stats.BackgroundJobs = len(s.analyticsQueue)
		mu.Unlock()
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if len(errors) > 0 {
		return nil, fmt.Errorf("failed to get complete statistics: %v", errors)
	}

	return stats, nil
}
