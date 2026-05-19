package repository

import (
	"database/sql"
	"encoding/json"

	"v2ray-dash/backend/internal/model"
)

type TemplateRepository struct {
	db *sql.DB
}

func NewTemplateRepository(db *sql.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(tmpl *model.Template) error {
	configJSON, err := json.Marshal(tmpl.Config)
	if err != nil {
		return err
	}
	return r.db.QueryRow(
		`INSERT INTO templates (name, description, config) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		tmpl.Name, tmpl.Description, configJSON,
	).Scan(&tmpl.ID, &tmpl.CreatedAt, &tmpl.UpdatedAt)
}

func (r *TemplateRepository) List() ([]*model.Template, error) {
	rows, err := r.db.Query(`SELECT id, name, description, config, created_at, updated_at FROM templates ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*model.Template
	for rows.Next() {
		var t model.Template
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &configJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(configJSON, &t.Config)
		templates = append(templates, &t)
	}
	if templates == nil {
		templates = []*model.Template{}
	}
	return templates, nil
}

func (r *TemplateRepository) GetByID(id string) (*model.Template, error) {
	var t model.Template
	var configJSON []byte
	err := r.db.QueryRow(`SELECT id, name, description, config, created_at, updated_at FROM templates WHERE id = $1`, id).Scan(&t.ID, &t.Name, &t.Description, &configJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(configJSON, &t.Config)
	return &t, nil
}

func (r *TemplateRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM templates WHERE id = $1`, id)
	return err
}