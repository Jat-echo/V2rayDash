package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	result, err := r.db.Exec(
		`INSERT INTO templates (name, description, config) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		tmpl.Name, tmpl.Description, configJSON,
	)
	if err != nil {
		return err
	}
	var id int
	var createdAt, updatedAt time.Time
	err = result.Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	tmpl.ID = fmt.Sprintf("%d", id)
	tmpl.CreatedAt = createdAt
	tmpl.UpdatedAt = updatedAt
	return nil
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
		var id int
		if err := rows.Scan(&id, &t.Name, &t.Description, &configJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.ID = fmt.Sprintf("%d", id)
		json.Unmarshal(configJSON, &t.Config)
		templates = append(templates, &t)
	}
	return templates, nil
}

func (r *TemplateRepository) GetByID(id string) (*model.Template, error) {
	var t model.Template
	var configJSON []byte
	var idInt int
	err := r.db.QueryRow(`SELECT id, name, description, config, created_at, updated_at FROM templates WHERE id = $1`, id).Scan(&idInt, &t.Name, &t.Description, &configJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.ID = fmt.Sprintf("%d", idInt)
	json.Unmarshal(configJSON, &t.Config)
	return &t, nil
}

func (r *TemplateRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM templates WHERE id = $1`, id)
	return err
}