package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	SessionCookieName = "soundhub_session"
	RoleAdmin         = "admin"
	RoleUser          = "user"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrRegistrationClosed = errors.New("registration is closed")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrForbidden          = errors.New("forbidden")
	ErrLastAdmin          = errors.New("cannot remove last admin")
)

type User struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"display_name"`
	Role          string    `json:"role"`
	Active        bool      `json:"active"`
	ProfilePublic bool      `json:"profile_public"`
	CreatedAt     time.Time `json:"created_at"`
}

type Session struct {
	ID        string
	UserID    string
	Token     string
	BootID    string
	ExpiresAt time.Time
	LastSeen  time.Time
	CreatedAt time.Time
}

type AuthStatus struct {
	SetupRequired       bool       `json:"setup_required"`
	RegistrationOpen    bool       `json:"registration_open"`
	Authenticated       bool       `json:"authenticated"`
	User                *User      `json:"user,omitempty"`
	SessionExpiresAt    *time.Time `json:"session_expires_at,omitempty"`
	SessionIdleTimeoutS int        `json:"session_idle_timeout_seconds"`
	SessionTTLSeconds   int        `json:"session_ttl_seconds"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type Service struct {
	repo        *Repository
	bootID      string
	idleTimeout time.Duration
	sessionTTL  time.Duration
}

func New(repo *Repository) *Service {
	return &Service{
		repo:        repo,
		bootID:      mustRandomID(16),
		idleTimeout: 30 * time.Minute,
		sessionTTL:  12 * time.Hour,
	}
}

func (s *Service) IdleTimeout() time.Duration {
	return s.idleTimeout
}

func (s *Service) SessionTTL() time.Duration {
	return s.sessionTTL
}

func (s *Service) Status(ctx context.Context, token string) (AuthStatus, error) {
	userCount, err := s.repo.UserCount(ctx)
	if err != nil {
		return AuthStatus{}, err
	}
	open, err := s.repo.RegistrationOpen(ctx)
	if err != nil {
		return AuthStatus{}, err
	}
	status := AuthStatus{
		SetupRequired:       userCount == 0,
		RegistrationOpen:    open || userCount == 0,
		SessionIdleTimeoutS: int(s.idleTimeout.Seconds()),
		SessionTTLSeconds:   int(s.sessionTTL.Seconds()),
	}
	if strings.TrimSpace(token) == "" {
		return status, nil
	}
	session, user, err := s.ValidateSession(ctx, token)
	if err != nil {
		return status, nil
	}
	status.Authenticated = true
	status.User = user
	status.SessionExpiresAt = &session.ExpiresAt
	return status, nil
}

func (s *Service) Register(ctx context.Context, username, displayName, password string) (User, Session, error) {
	username = normalizeUsername(username)
	displayName = strings.TrimSpace(displayName)
	password = strings.TrimSpace(password)
	if username == "" || displayName == "" || password == "" {
		return User{}, Session{}, ErrInvalidCredentials
	}

	userCount, err := s.repo.UserCount(ctx)
	if err != nil {
		return User{}, Session{}, err
	}
	open, err := s.repo.RegistrationOpen(ctx)
	if err != nil {
		return User{}, Session{}, err
	}
	if userCount > 0 && !open {
		return User{}, Session{}, ErrRegistrationClosed
	}
	existing, err := s.repo.UserByUsername(ctx, username)
	if err == nil && existing.ID != "" {
		return User{}, Session{}, ErrUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, Session{}, err
	}

	salt := mustRandomID(16)
	role := RoleUser
	if userCount == 0 {
		role = RoleAdmin
	}
	user := User{
		ID:          mustRandomID(16),
		Username:    username,
		DisplayName: displayName,
		Role:        role,
		Active:      true,
	}
	if err := s.repo.CreateUser(ctx, user, salt, hashPassword(password, salt)); err != nil {
		if isUniqueViolation(err) {
			return User{}, Session{}, ErrUserExists
		}
		return User{}, Session{}, err
	}
	created, err := s.repo.UserByUsername(ctx, username)
	if err != nil {
		return User{}, Session{}, err
	}
	session, err := s.createSession(ctx, created.ID)
	if err != nil {
		return User{}, Session{}, err
	}
	return created, session, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (User, Session, error) {
	username = normalizeUsername(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return User{}, Session{}, ErrInvalidCredentials
	}
	user, salt, hash, err := s.repo.UserAuthByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, Session{}, ErrInvalidCredentials
		}
		return User{}, Session{}, err
	}
	if !user.Active {
		return User{}, Session{}, ErrForbidden
	}
	if hashPassword(password, salt) != hash {
		return User{}, Session{}, ErrInvalidCredentials
	}
	session, err := s.createSession(ctx, user.ID)
	if err != nil {
		return User{}, Session{}, err
	}
	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return s.repo.DeleteSessionByToken(ctx, token)
}

func (s *Service) ValidateSession(ctx context.Context, token string) (Session, *User, error) {
	session, user, err := s.repo.SessionWithUser(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, nil, ErrUnauthorized
		}
		return Session{}, nil, err
	}
	now := time.Now()
	if session.BootID != s.bootID || now.After(session.ExpiresAt) || now.Sub(session.LastSeen) > s.idleTimeout || !user.Active {
		_ = s.repo.DeleteSessionByToken(ctx, token)
		return Session{}, nil, ErrUnauthorized
	}
	if err := s.repo.TouchSession(ctx, session.ID, now); err != nil {
		return Session{}, nil, err
	}
	session.LastSeen = now
	return session, user, nil
}

func (s *Service) ListUsers(ctx context.Context, current *User) ([]User, bool, error) {
	if current == nil || current.Role != RoleAdmin {
		return nil, false, ErrForbidden
	}
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, false, err
	}
	open, err := s.repo.RegistrationOpen(ctx)
	if err != nil {
		return nil, false, err
	}
	return users, open, nil
}

func (s *Service) ListShareableUsers(ctx context.Context, current *User) ([]User, error) {
	if current == nil {
		return nil, ErrUnauthorized
	}
	users, err := s.repo.ListActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]User, 0, len(users))
	for _, user := range users {
		if user.ID == current.ID {
			continue
		}
		filtered = append(filtered, user)
	}
	return filtered, nil
}

func (s *Service) SetRegistrationOpen(ctx context.Context, current *User, open bool) error {
	if current == nil || current.Role != RoleAdmin {
		return ErrForbidden
	}
	return s.repo.SetRegistrationOpen(ctx, open)
}

func (s *Service) SetUserActive(ctx context.Context, current *User, userID string, active bool) error {
	if current == nil || current.Role != RoleAdmin {
		return ErrForbidden
	}
	if current.ID == userID && !active {
		return ErrForbidden
	}
	target, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if target.Role == RoleAdmin && !active {
		adminCount, err := s.repo.ActiveAdminCount(ctx)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}
	if err := s.repo.SetUserActive(ctx, userID, active); err != nil {
		return err
	}
	if !active {
		_ = s.repo.DeleteSessionsByUser(ctx, userID)
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, current *User, userID string) error {
	if current == nil || current.Role != RoleAdmin {
		return ErrForbidden
	}
	target, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if target.Role == RoleAdmin {
		adminCount, err := s.repo.ActiveAdminCount(ctx)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}
	if current.ID == userID {
		return ErrForbidden
	}
	return s.repo.DeleteUser(ctx, userID)
}

func (s *Service) GetProfile(ctx context.Context, current *User, userID string) (User, error) {
	if current == nil {
		return User{}, ErrUnauthorized
	}
	target, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if current.Role == RoleAdmin || current.ID == target.ID || target.ProfilePublic {
		return target, nil
	}
	return User{}, ErrForbidden
}

func (s *Service) UpdateProfile(ctx context.Context, current *User, username, displayName string, profilePublic bool) (User, error) {
	if current == nil {
		return User{}, ErrUnauthorized
	}
	username = normalizeUsername(username)
	displayName = strings.TrimSpace(displayName)
	if username == "" || displayName == "" {
		return User{}, ErrInvalidCredentials
	}
	existing, err := s.repo.UserByUsername(ctx, username)
	if err == nil && existing.ID != "" && existing.ID != current.ID {
		return User{}, ErrUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	if err := s.repo.UpdateUserProfile(ctx, current.ID, username, displayName, profilePublic); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return s.repo.UserByID(ctx, current.ID)
}

func (s *Service) ChangePassword(ctx context.Context, current *User, currentPassword, nextPassword string) error {
	if current == nil {
		return ErrUnauthorized
	}
	currentPassword = strings.TrimSpace(currentPassword)
	nextPassword = strings.TrimSpace(nextPassword)
	if currentPassword == "" || nextPassword == "" {
		return ErrInvalidCredentials
	}
	user, salt, hash, err := s.repo.UserAuthByUsername(ctx, current.Username)
	if err != nil {
		return err
	}
	if user.ID != current.ID || hashPassword(currentPassword, salt) != hash {
		return ErrInvalidCredentials
	}
	nextSalt := mustRandomID(16)
	return s.repo.UpdateUserPassword(ctx, current.ID, nextSalt, hashPassword(nextPassword, nextSalt))
}

func (s *Service) createSession(ctx context.Context, userID string) (Session, error) {
	now := time.Now()
	session := Session{
		ID:        mustRandomID(16),
		UserID:    userID,
		Token:     mustRandomID(32),
		BootID:    s.bootID,
		CreatedAt: now,
		LastSeen:  now,
		ExpiresAt: now.Add(s.sessionTTL),
	}
	return session, s.repo.CreateSession(ctx, session)
}

func (r *Repository) UserCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_users`).Scan(&count)
	return count, err
}

func (r *Repository) RegistrationOpen(ctx context.Context) (bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = 'registration_open'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return strings.EqualFold(strings.TrimSpace(value), "true"), err
}

func (r *Repository) SetRegistrationOpen(ctx context.Context, open bool) error {
	value := "false"
	if open {
		value = "true"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('registration_open', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, value)
	return err
}

func (r *Repository) CreateUser(ctx context.Context, user User, salt, hash string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_users (id, username, display_name, password_salt, password_hash, role, active, profile_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`, user.ID, user.Username, user.DisplayName, salt, hash, user.Role, user.Active, true)
	return err
}

func (r *Repository) UserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, role, active, profile_public, created_at
		FROM app_users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt)
	return user, err
}

func (r *Repository) UserAuthByUsername(ctx context.Context, username string) (User, string, string, error) {
	var user User
	var salt, hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, role, active, profile_public, created_at, password_salt, password_hash
		FROM app_users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt, &salt, &hash)
	return user, salt, hash, err
}

func (r *Repository) UserByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, role, active, profile_public, created_at
		FROM app_users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt)
	return user, err
}

func (r *Repository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, display_name, role, active, profile_public, created_at
		FROM app_users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) ListActiveUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, username, display_name, role, active, profile_public, created_at
		FROM app_users
		WHERE active = TRUE
		ORDER BY display_name ASC, username ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) ActiveAdminCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_users WHERE role = $1 AND active = TRUE`, RoleAdmin).Scan(&count)
	return count, err
}

func (r *Repository) SetUserActive(ctx context.Context, userID string, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE app_users SET active = $2, updated_at = NOW() WHERE id = $1`, userID, active)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM app_users WHERE id = $1`, userID)
	return err
}

func (r *Repository) UpdateUserProfile(ctx context.Context, userID, username, displayName string, profilePublic bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE app_users
		SET username = $2, display_name = $3, profile_public = $4, updated_at = NOW()
		WHERE id = $1
	`, userID, username, displayName, profilePublic)
	return err
}

func (r *Repository) UpdateUserPassword(ctx context.Context, userID, salt, hash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE app_users
		SET password_salt = $2, password_hash = $3, updated_at = NOW()
		WHERE id = $1
	`, userID, salt, hash)
	return err
}

func (r *Repository) CreateSession(ctx context.Context, session Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (id, user_id, token, boot_id, expires_at, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.UserID, session.Token, session.BootID, session.ExpiresAt, session.LastSeen, session.CreatedAt)
	return err
}

func (r *Repository) SessionWithUser(ctx context.Context, token string) (Session, *User, error) {
	var session Session
	var user User
	err := r.db.QueryRowContext(ctx, `
		SELECT s.id, s.user_id, s.token, s.boot_id, s.expires_at, s.last_seen_at, s.created_at,
		       u.id, u.username, u.display_name, u.role, u.active, u.profile_public, u.created_at
		FROM auth_sessions s
		JOIN app_users u ON u.id = s.user_id
		WHERE s.token = $1
	`, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.BootID, &session.ExpiresAt, &session.LastSeen, &session.CreatedAt,
		&user.ID, &user.Username, &user.DisplayName, &user.Role, &user.Active, &user.ProfilePublic, &user.CreatedAt,
	)
	if err != nil {
		return Session{}, nil, err
	}
	return session, &user, nil
}

func (r *Repository) TouchSession(ctx context.Context, sessionID string, seenAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE auth_sessions SET last_seen_at = $2 WHERE id = $1`, sessionID, seenAt)
	return err
}

func (r *Repository) DeleteSessionByToken(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token = $1`, token)
	return err
}

func (r *Repository) DeleteSessionsByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id = $1`, userID)
	return err
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hashPassword(password, salt string) string {
	payload := []byte(salt + ":" + password)
	sum := sha256.Sum256(payload)
	for i := 0; i < 120000; i += 1 {
		next := sha256.Sum256(sum[:])
		sum = next
	}
	return hex.EncodeToString(sum[:])
}

func mustRandomID(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique")
}
