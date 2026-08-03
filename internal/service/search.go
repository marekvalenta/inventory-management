package service

import (
	"database/sql"
	"fmt"
	"strings"
)

type SearchParams struct {
	Query string
	Type  string
	Limit int
}

type LocationResult struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ParentID   *string `json:"parent_id"`
	ParentName *string `json:"parent_name"`
}

type DefinitionResult struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Unit          *string `json:"unit"`
	ParentDefName *string `json:"parent_def_name"`
}

type StackResult struct {
	DefinitionID       string  `json:"definition_id"`
	DefinitionName     string  `json:"definition_name"`
	Unit               *string `json:"unit"`
	LocationID         *string `json:"location_id"`
	LocationName       *string `json:"location_name"`
	ParentInstanceID   *string `json:"parent_instance_id"`
	ParentInstanceName *string `json:"parent_instance_name"`
	TotalQuantity      int     `json:"total_quantity"`
	InstanceCount      int     `json:"instance_count"`
	SingleInstanceID   *string `json:"single_instance_id,omitempty"`
}

type TotalCounts struct {
	Locations   int `json:"locations"`
	Definitions int `json:"definitions"`
	Stacks      int `json:"stacks"`
}

type SearchResponse struct {
	Locations   []LocationResult   `json:"locations,omitempty"`
	Definitions []DefinitionResult `json:"definitions,omitempty"`
	Stacks      []StackResult      `json:"stacks,omitempty"`
	TotalCounts TotalCounts        `json:"total_counts"`
}

type SearchService struct {
	db *sql.DB
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{db: db}
}

func (s *SearchService) Search(params SearchParams) (*SearchResponse, error) {
	query := strings.TrimSpace(params.Query)
	if len(query) < 2 {
		return nil, fmt.Errorf("Search term must be at least 2 characters: %w", ErrInvalidInput)
	}
	if len(query) > 200 {
		return nil, fmt.Errorf("Search term must be at most 200 characters: %w", ErrInvalidInput)
	}

	validTypes := map[string]bool{"all": true, "locations": true, "definitions": true, "stacks": true}
	if !validTypes[params.Type] {
		params.Type = "all"
	}

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	likePattern := "%" + query + "%"
	exactMatch := query
	startsWith := query + "%"

	resp := &SearchResponse{
		TotalCounts: TotalCounts{},
	}

	var locCount, defCount, stackCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM locations WHERE name LIKE ?`, likePattern).Scan(&locCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM item_definitions WHERE name LIKE ?`, likePattern).Scan(&defCount)

	stackCountRow := s.db.QueryRow(`
		SELECT COUNT(DISTINCT d.id || COALESCE(i.location_id, '') || COALESCE(i.parent_instance_id, ''))
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		WHERE d.name LIKE ?
	`, likePattern)
	stackCountRow.Scan(&stackCount)

	resp.TotalCounts = TotalCounts{
		Locations:   locCount,
		Definitions: defCount,
		Stacks:      stackCount,
	}

	if params.Type == "all" || params.Type == "locations" {
		rows, err := s.db.Query(`
			SELECT l.id, l.name, l.parent_id, p.name AS parent_name
			FROM locations l
			LEFT JOIN locations p ON p.id = l.parent_id
			WHERE l.name LIKE ?
			ORDER BY
				CASE WHEN l.name = ? THEN 0 WHEN l.name LIKE ? THEN 1 ELSE 2 END,
				l.name ASC
			LIMIT ?
		`, likePattern, exactMatch, startsWith, limit)
		if err != nil {
			return nil, fmt.Errorf("search locations: %w", err)
		}
		defer rows.Close()

		var locations []LocationResult
		for rows.Next() {
			var lr LocationResult
			var parentID, parentName sql.NullString
			if err := rows.Scan(&lr.ID, &lr.Name, &parentID, &parentName); err != nil {
				return nil, fmt.Errorf("scan location result: %w", err)
			}
			if parentID.Valid {
				lr.ParentID = &parentID.String
				lr.ParentName = &parentName.String
			}
			locations = append(locations, lr)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if locations == nil {
			locations = []LocationResult{}
		}
		resp.Locations = locations
	}

	if params.Type == "all" || params.Type == "definitions" {
		rows, err := s.db.Query(`
			SELECT d.id, d.name, d.unit, pd.name AS parent_def_name
			FROM item_definitions d
			LEFT JOIN item_definitions pd ON pd.id = d.parent_def_id
			WHERE d.name LIKE ?
			ORDER BY
				CASE WHEN d.name = ? THEN 0 WHEN d.name LIKE ? THEN 1 ELSE 2 END,
				d.name ASC
			LIMIT ?
		`, likePattern, exactMatch, startsWith, limit)
		if err != nil {
			return nil, fmt.Errorf("search definitions: %w", err)
		}
		defer rows.Close()

		var definitions []DefinitionResult
		for rows.Next() {
			var dr DefinitionResult
			var unit, parentDefName sql.NullString
			if err := rows.Scan(&dr.ID, &dr.Name, &unit, &parentDefName); err != nil {
				return nil, fmt.Errorf("scan definition result: %w", err)
			}
			if unit.Valid {
				dr.Unit = &unit.String
			}
			if parentDefName.Valid {
				dr.ParentDefName = &parentDefName.String
			}
			definitions = append(definitions, dr)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if definitions == nil {
			definitions = []DefinitionResult{}
		}
		resp.Definitions = definitions
	}

	if params.Type == "all" || params.Type == "stacks" {
		rows, err := s.db.Query(`
			SELECT
				d.id AS definition_id,
				d.name AS definition_name,
				d.unit,
				i.location_id,
				l.name AS location_name,
				i.parent_instance_id,
				pi_def.name AS parent_instance_name,
				COALESCE(SUM(i.quantity), 0) AS total_quantity,
				COUNT(i.id) AS instance_count,
				MIN(i.id) AS first_instance_id
			FROM item_instances i
			JOIN item_definitions d ON d.id = i.definition_id
			LEFT JOIN locations l ON l.id = i.location_id
			LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
			LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
			WHERE d.name LIKE ?
			GROUP BY d.id, i.location_id, i.parent_instance_id
			ORDER BY
				CASE WHEN d.name = ? THEN 0 WHEN d.name LIKE ? THEN 1 ELSE 2 END,
				d.name ASC
			LIMIT ?
		`, likePattern, exactMatch, startsWith, limit)
		if err != nil {
			return nil, fmt.Errorf("search stacks: %w", err)
		}
		defer rows.Close()

		var stacks []StackResult
		for rows.Next() {
			var sr StackResult
			var unit, locID, locName, parentInstID, parentInstName sql.NullString
			var firstInstanceID sql.NullString
			if err := rows.Scan(
				&sr.DefinitionID, &sr.DefinitionName, &unit,
				&locID, &locName, &parentInstID, &parentInstName,
				&sr.TotalQuantity, &sr.InstanceCount,
				&firstInstanceID,
			); err != nil {
				return nil, fmt.Errorf("scan stack result: %w", err)
			}
			if unit.Valid {
				sr.Unit = &unit.String
			}
			if locID.Valid {
				sr.LocationID = &locID.String
				sr.LocationName = &locName.String
			}
			if parentInstID.Valid {
				sr.ParentInstanceID = &parentInstID.String
				sr.ParentInstanceName = &parentInstName.String
			}
			if firstInstanceID.Valid && sr.InstanceCount == 1 {
				sr.SingleInstanceID = &firstInstanceID.String
			}
			stacks = append(stacks, sr)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if stacks == nil {
			stacks = []StackResult{}
		}
		resp.Stacks = stacks
	}

	return resp, nil
}
