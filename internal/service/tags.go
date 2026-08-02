package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Tag struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Color                 *string `json:"color"`
	LinkedDefinitionsCount int    `json:"linked_definitions_count"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
}

type TagService struct {
	db *sql.DB
}

func NewTagService(db *sql.DB) *TagService {
	return &TagService{db: db}
}

func (s *TagService) GetAll() ([]Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.color,
			(SELECT COUNT(*) FROM definition_tags dt WHERE dt.tag_id = t.id) AS linked_definitions_count,
			t.created_at, t.updated_at
		FROM tags t
		ORDER BY t.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.LinkedDefinitionsCount, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if tags == nil {
		tags = []Tag{}
	}

	return tags, rows.Err()
}

func (s *TagService) GetByID(id string) (*Tag, error) {
	var tag Tag
	err := s.db.QueryRow(`
		SELECT t.id, t.name, t.color,
			(SELECT COUNT(*) FROM definition_tags dt WHERE dt.tag_id = t.id) AS linked_definitions_count,
			t.created_at, t.updated_at
		FROM tags t
		WHERE t.id = ?
	`, id).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.LinkedDefinitionsCount, &tag.CreatedAt, &tag.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}

	return &tag, nil
}

func (s *TagService) Create(name string, color *string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return nil, fmt.Errorf("name must be between 2 and 100 characters: %w", ErrInvalidInput)
	}

	if color != nil {
		trimmed := strings.TrimSpace(*color)
		if len(trimmed) > 10 {
			return nil, fmt.Errorf("color must be at most 10 characters: %w", ErrInvalidInput)
		}
		if trimmed == "" {
			color = nil
		} else {
			color = &trimmed
		}
	}

	id := uuid.New().String()

	_, err := s.db.Exec(
		`INSERT INTO tags (id, name, color) VALUES (?, ?, ?)`,
		id, name, color,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("Tag '%s' already exists", name)
		}
		return nil, fmt.Errorf("create tag: %w", err)
	}

	return s.GetByID(id)
}

func (s *TagService) Update(id string, name *string, color *string) (*Tag, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if len(trimmed) < 2 || len(trimmed) > 100 {
			return nil, fmt.Errorf("name must be between 2 and 100 characters: %w", ErrInvalidInput)
		}
		*name = trimmed
	} else {
		name = &existing.Name
	}

	if color != nil {
		trimmed := strings.TrimSpace(*color)
		if len(trimmed) > 10 {
			return nil, fmt.Errorf("color must be at most 10 characters: %w", ErrInvalidInput)
		}
		if trimmed == "" {
			color = nil
		} else {
			color = &trimmed
		}
	} else {
		color = existing.Color
	}

	_, err = s.db.Exec(
		`UPDATE tags SET name = ?, color = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, color, id,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("Tag '%s' already exists", *name)
		}
		return nil, fmt.Errorf("update tag: %w", err)
	}

	return s.GetByID(id)
}

func (s *TagService) Delete(id string) (int, error) {
	var linkedCount int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM definition_tags WHERE tag_id = ?`,
		id,
	).Scan(&linkedCount)
	if err != nil {
		return 0, fmt.Errorf("count linked definitions: %w", err)
	}

	result, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return 0, fmt.Errorf("delete tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return 0, ErrNotFound
	}

	return linkedCount, nil
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
