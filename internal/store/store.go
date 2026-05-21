package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrProfileExists   = errors.New("profile already exists")
	ErrProfileNotFound = errors.New("profile not found")
)

type Profile struct {
	Wallet      string  `json:"wallet"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initializeSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) initializeSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store not initialized")
	}

	const schema = `
CREATE TABLE IF NOT EXISTS profiles (
	wallet TEXT PRIMARY KEY,
	display_name TEXT NOT NULL,
	avatar_url TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_profiles_display_name ON profiles(display_name);
`

	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}

	return nil
}

func (s *Store) CreateProfile(ctx context.Context, wallet, displayName string, avatarURL *string) (*Profile, error) {
	wallet = strings.TrimSpace(wallet)
	displayName = strings.TrimSpace(displayName)
	if wallet == "" || displayName == "" {
		return nil, fmt.Errorf("wallet and display_name are required")
	}

	avatar := normalizeAvatarURL(avatarURL)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO profiles (wallet, display_name, avatar_url)
		VALUES (?, ?, ?)
	`, wallet, displayName, avatar)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrProfileExists
		}
		return nil, err
	}

	return s.GetProfile(ctx, wallet)
}

func (s *Store) GetProfile(ctx context.Context, wallet string) (*Profile, error) {
	wallet = strings.TrimSpace(wallet)
	if wallet == "" {
		return nil, fmt.Errorf("wallet is required")
	}

	var profile Profile
	var avatar sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT wallet, display_name, avatar_url, created_at, updated_at
		FROM profiles
		WHERE wallet = ?
	`, wallet).Scan(
		&profile.Wallet,
		&profile.DisplayName,
		&avatar,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	if avatar.Valid {
		avatarValue := avatar.String
		profile.AvatarURL = &avatarValue
	}

	return &profile, nil
}

func (s *Store) UpdateProfile(ctx context.Context, wallet string, displayName, avatarURL *string) (*Profile, error) {
	wallet = strings.TrimSpace(wallet)
	if wallet == "" {
		return nil, fmt.Errorf("wallet is required")
	}

	setClauses := []string{}
	args := []any{}

	if displayName != nil {
		trimmed := strings.TrimSpace(*displayName)
		if trimmed == "" {
			return nil, fmt.Errorf("display_name cannot be empty")
		}
		setClauses = append(setClauses, "display_name = ?")
		args = append(args, trimmed)
	}

	if avatarURL != nil {
		setClauses = append(setClauses, "avatar_url = ?")
		args = append(args, normalizeAvatarURL(avatarURL))
	}

	if len(setClauses) == 0 {
		return s.GetProfile(ctx, wallet)
	}

	setClauses = append(setClauses, "updated_at = datetime('now')")
	args = append(args, wallet)

	query := fmt.Sprintf("UPDATE profiles SET %s WHERE wallet = ?", strings.Join(setClauses, ", "))
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrProfileNotFound
	}

	return s.GetProfile(ctx, wallet)
}

func (s *Store) GetProfilesByWallets(ctx context.Context, wallets []string) ([]Profile, error) {
	cleanWallets := dedupeWallets(wallets)
	if len(cleanWallets) == 0 {
		return []Profile{}, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(cleanWallets)), ",")
	args := make([]any, 0, len(cleanWallets))
	for _, wallet := range cleanWallets {
		args = append(args, wallet)
	}

	query := fmt.Sprintf(`
		SELECT wallet, display_name, avatar_url, created_at, updated_at
		FROM profiles
		WHERE wallet IN (%s)
	`, placeholders)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]Profile, 0, len(cleanWallets))
	for rows.Next() {
		var profile Profile
		var avatar sql.NullString
		if err := rows.Scan(
			&profile.Wallet,
			&profile.DisplayName,
			&avatar,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if avatar.Valid {
			avatarValue := avatar.String
			profile.AvatarURL = &avatarValue
		}
		profiles = append(profiles, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func dedupeWallets(wallets []string) []string {
	seen := make(map[string]struct{}, len(wallets))
	unique := make([]string, 0, len(wallets))
	for _, wallet := range wallets {
		trimmed := strings.TrimSpace(wallet)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func normalizeAvatarURL(avatarURL *string) any {
	if avatarURL == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*avatarURL)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
