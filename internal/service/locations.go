package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Location struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type TreeNode struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Children    []TreeNode `json:"children"`
}

type Contents struct {
	SubLocations []Location        `json:"sub_locations"`
	Instances    []InstanceSummary `json:"instances"`
}

type InstanceSummary struct {
	ID                 string  `json:"id"`
	DefinitionID       string  `json:"definition_id"`
	DefinitionName     string  `json:"definition_name"`
	Quantity           int     `json:"quantity"`
	LocationID         *string `json:"location_id"`
	LocationName       *string `json:"location_name"`
	ParentInstanceID   *string `json:"parent_instance_id"`
	ParentInstanceName *string `json:"parent_instance_name"`
	UpdatedAt          string  `json:"updated_at"`
}

type BreadcrumbNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LocationService struct {
	db *sql.DB
}

func NewLocationService(db *sql.DB) *LocationService {
	return &LocationService{db: db}
}

func (s *LocationService) List(parentID *string) ([]Location, error) {
	var query string
	var args []interface{}

	if parentID != nil {
		query = `SELECT id, name, description, parent_id, created_at, updated_at
		         FROM locations WHERE parent_id = ? ORDER BY name ASC`
		args = append(args, *parentID)
	} else {
		query = `SELECT id, name, description, parent_id, created_at, updated_at
		         FROM locations ORDER BY name ASC`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.ParentID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		locations = append(locations, l)
	}

	if locations == nil {
		locations = []Location{}
	}

	return locations, rows.Err()
}

func (s *LocationService) GetByID(id string) (*Location, error) {
	var l Location
	err := s.db.QueryRow(
		`SELECT id, name, description, parent_id, created_at, updated_at
		 FROM locations WHERE id = ?`, id,
	).Scan(&l.ID, &l.Name, &l.Description, &l.ParentID, &l.CreatedAt, &l.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get location: %w", err)
	}

	return &l, nil
}

func (s *LocationService) GetTree() ([]TreeNode, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, parent_id FROM locations ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}
	defer rows.Close()

	nodeMap := make(map[string]*TreeNode)
	var roots []TreeNode

	for rows.Next() {
		var id, name string
		var description, parentID *string
		if err := rows.Scan(&id, &name, &description, &parentID); err != nil {
			return nil, fmt.Errorf("scan tree node: %w", err)
		}

		node := TreeNode{
			ID:          id,
			Name:        name,
			Description: description,
			Children:    []TreeNode{},
		}
		nodeMap[id] = &node

		if parentID == nil {
			roots = append(roots, node)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows2, err := s.db.Query(
		`SELECT id, name, description, parent_id FROM locations WHERE parent_id IS NOT NULL ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("get tree children: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, name string
		var description, parentID *string
		if err := rows2.Scan(&id, &name, &description, &parentID); err != nil {
			return nil, fmt.Errorf("scan tree child: %w", err)
		}

		if parent, ok := nodeMap[*parentID]; ok {
			if child, ok := nodeMap[id]; ok {
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	if roots == nil {
		roots = []TreeNode{}
	}

	return roots, rows2.Err()
}

func (s *LocationService) GetChildren(parentID string) ([]Location, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, parent_id, created_at, updated_at
		 FROM locations WHERE parent_id = ? ORDER BY name ASC`, parentID,
	)
	if err != nil {
		return nil, fmt.Errorf("get children: %w", err)
	}
	defer rows.Close()

	var children []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.ParentID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}
		children = append(children, l)
	}

	if children == nil {
		children = []Location{}
	}

	return children, rows.Err()
}

func (s *LocationService) GetContents(id string) (*Contents, error) {
	subLocations, err := s.GetChildren(id)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT i.id, i.definition_id, d.name, i.quantity
		 FROM item_instances i
		 JOIN item_definitions d ON d.id = i.definition_id
		 WHERE i.location_id = ?
		 ORDER BY d.name ASC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get contents instances: %w", err)
	}
	defer rows.Close()

	var instances []InstanceSummary
	for rows.Next() {
		var inst InstanceSummary
		if err := rows.Scan(&inst.ID, &inst.DefinitionID, &inst.DefinitionName, &inst.Quantity); err != nil {
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		instances = append(instances, inst)
	}

	if instances == nil {
		instances = []InstanceSummary{}
	}

	return &Contents{
		SubLocations: subLocations,
		Instances:    instances,
	}, rows.Err()
}

func (s *LocationService) GetBreadcrumb(id string) ([]BreadcrumbNode, error) {
	rows, err := s.db.Query(`
		WITH RECURSIVE ancestors AS (
			SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
			UNION ALL
			SELECT l.id, l.name, l.parent_id, a.depth + 1
			FROM locations l JOIN ancestors a ON l.id = a.parent_id
		)
		SELECT id, name FROM ancestors ORDER BY depth DESC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get breadcrumb: %w", err)
	}
	defer rows.Close()

	var breadcrumb []BreadcrumbNode
	for rows.Next() {
		var node BreadcrumbNode
		if err := rows.Scan(&node.ID, &node.Name); err != nil {
			return nil, fmt.Errorf("scan breadcrumb: %w", err)
		}
		breadcrumb = append(breadcrumb, node)
	}

	if len(breadcrumb) == 0 {
		return nil, ErrNotFound
	}

	return breadcrumb, rows.Err()
}

func (s *LocationService) Create(name string, description *string, parentID *string) (*Location, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 200 {
		return nil, fmt.Errorf("name must be between 2 and 200 characters: %w", ErrInvalidInput)
	}

	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if len(trimmed) > 2000 {
			return nil, fmt.Errorf("description must be at most 2000 characters: %w", ErrInvalidInput)
		}
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}

	if parentID != nil {
		if _, err := s.GetByID(*parentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, fmt.Errorf("parent not found: %w", ErrInvalidInput)
			}
			return nil, err
		}
	}

	id := uuid.New().String()

	_, err := s.db.Exec(
		`INSERT INTO locations (id, name, description, parent_id) VALUES (?, ?, ?, ?)`,
		id, name, description, parentID,
	)
	if err != nil {
		return nil, fmt.Errorf("create location: %w", err)
	}

	return s.GetByID(id)
}

func (s *LocationService) Update(id string, name *string, description *string, parentID *string) (*Location, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if len(trimmed) < 2 || len(trimmed) > 200 {
			return nil, fmt.Errorf("name must be between 2 and 200 characters: %w", ErrInvalidInput)
		}
		*name = trimmed
	} else {
		name = &existing.Name
	}

	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if len(trimmed) > 2000 {
			return nil, fmt.Errorf("description must be at most 2000 characters: %w", ErrInvalidInput)
		}
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	} else {
		description = existing.Description
	}

	if parentID != nil && *parentID == "" {
		parentID = nil
	}

	if parentID != existing.ParentID && (parentID == nil || existing.ParentID == nil || *parentID != *existing.ParentID) {
		rootID, err := s.GetRootID()
		if err != nil {
			return nil, err
		}

		if parentID != nil && *parentID == id {
			return nil, fmt.Errorf("cannot set parent to self: %w", ErrInvalidInput)
		}

		if id == rootID {
			return nil, fmt.Errorf("cannot reparent root location: %w", ErrInvalidInput)
		}

		if parentID != nil {
			if _, err := s.GetByID(*parentID); err != nil {
				if errors.Is(err, ErrNotFound) {
					return nil, fmt.Errorf("parent not found: %w", ErrInvalidInput)
				}
				return nil, err
			}

			isDescendant, err := s.isDescendant(id, *parentID)
			if err != nil {
				return nil, err
			}
			if isDescendant {
				return nil, fmt.Errorf("cannot move location into its own subtree: %w", ErrInvalidInput)
			}
		}
	} else if parentID == nil && existing.ParentID != nil {
		rootID, err := s.GetRootID()
		if err != nil {
			return nil, err
		}
		if id == rootID {
			return nil, fmt.Errorf("cannot reparent root location: %w", ErrInvalidInput)
		}
	} else if parentID != nil {
		if *parentID == id {
			return nil, fmt.Errorf("cannot set parent to self: %w", ErrInvalidInput)
		}
	}

	_, err = s.db.Exec(
		`UPDATE locations SET name = ?, description = ?, parent_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, description, parentID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update location: %w", err)
	}

	return s.GetByID(id)
}

func (s *LocationService) Delete(id string) (*DeleteBlock, error) {
	rootID, err := s.GetRootID()
	if err != nil {
		return nil, err
	}

	if id == rootID {
		return nil, fmt.Errorf("cannot delete root location: %w", ErrInvalidInput)
	}

	var childCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM locations WHERE parent_id = ?`, id,
	).Scan(&childCount); err != nil {
		return nil, fmt.Errorf("count children: %w", err)
	}

	var instanceCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM item_instances WHERE location_id = ?`, id,
	).Scan(&instanceCount); err != nil {
		return nil, fmt.Errorf("count instances: %w", err)
	}

	if childCount > 0 || instanceCount > 0 {
		return &DeleteBlock{
			ChildCount:    childCount,
			InstanceCount: instanceCount,
		}, ErrConflict
	}

	_, err = s.db.Exec(`DELETE FROM locations WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("delete location: %w", err)
	}

	return nil, nil
}

type DeleteBlock struct {
	ChildCount    int `json:"child_count"`
	InstanceCount int `json:"instance_count"`
}

func (s *LocationService) GetRootID() (string, error) {
	var rootID string
	err := s.db.QueryRow(`SELECT root_location_id FROM settings WHERE id = 1`).Scan(&rootID)
	if err != nil {
		return "", fmt.Errorf("get root location id: %w", err)
	}
	return rootID, nil
}

func (s *LocationService) isDescendant(id, potentialAncestor string) (bool, error) {
	rows, err := s.db.Query(`
		WITH RECURSIVE descendants AS (
			SELECT id FROM locations WHERE parent_id = ?
			UNION ALL
			SELECT l.id FROM locations l JOIN descendants d ON l.parent_id = d.id
		)
		SELECT id FROM descendants WHERE id = ? LIMIT 1
	`, id, potentialAncestor)
	if err != nil {
		return false, fmt.Errorf("check descendant: %w", err)
	}
	defer rows.Close()

	return rows.Next(), rows.Err()
}
