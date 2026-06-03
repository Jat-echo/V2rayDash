package repository

import (
	"database/sql"
	"time"
)

type AdminUser struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CountAdmins() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count)
	return count, err
}

func (r *AuthRepository) GetByUsername(username string) (*AdminUser, error) {
	var u AdminUser
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, created_at FROM admin_users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AuthRepository) Create(username, passwordHash string) error {
	_, err := r.db.Exec(
		`INSERT INTO admin_users (username, password_hash) VALUES ($1, $2)`,
		username, passwordHash,
	)
	return err
}

func (r *AuthRepository) UpdatePassword(id, passwordHash string) error {
	_, err := r.db.Exec(
		`UPDATE admin_users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		passwordHash, id,
	)
	return err
}
