package repository

import (
	"database/sql"
	"v2ray-dash/backend/internal/model"
)

type SettingRepository struct {
	db *sql.DB
}

func NewSettingRepository(db *sql.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) Get(id string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	err := r.db.QueryRow(
		"SELECT id, value, description, updated_at FROM system_settings WHERE id = $1",
		id,
	).Scan(&setting.ID, &setting.Value, &setting.Description, &setting.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *SettingRepository) Update(id, value string) error {
	_, err := r.db.Exec(
		"UPDATE system_settings SET value = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		value, id,
	)
	return err
}
