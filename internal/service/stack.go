package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type BrowseStack struct {
	DefinitionID       string  `json:"definition_id"`
	DefinitionName     string  `json:"definition_name"`
	Unit               *string `json:"unit"`
	LocationID         *string `json:"location_id"`
	LocationName       *string `json:"location_name"`
	ParentInstanceID   *string `json:"parent_instance_id"`
	ParentInstanceName *string `json:"parent_instance_name"`
	TotalQuantity      int     `json:"total_quantity"`
	InstanceCount      int     `json:"instance_count"`
	IsContainer        bool    `json:"is_container"`
	ChildCount         int     `json:"child_count"`
}

type StackListResult struct {
	Stacks     []BrowseStack `json:"stacks"`
	TotalCount int           `json:"total_count"`
	Truncated  bool          `json:"truncated,omitempty"`
}

type StackDetail struct {
	DefinitionID       string             `json:"definition_id"`
	DefinitionName     string             `json:"definition_name"`
	Unit               *string            `json:"unit"`
	IsContainer        bool               `json:"is_container"`
	ParentDefID        *string            `json:"parent_def_id"`
	ParentDefName      *string            `json:"parent_def_name"`
	LocationID         *string            `json:"location_id"`
	LocationName       *string            `json:"location_name"`
	ParentInstanceID   *string            `json:"parent_instance_id"`
	ParentInstanceName *string            `json:"parent_instance_name"`
	TotalQuantity      int                `json:"total_quantity"`
	InstanceCount      int                `json:"instance_count"`
	ChildCount         int                `json:"child_count"`
	Breadcrumb         []BreadcrumbEntry  `json:"breadcrumb"`
	Instances          []InstanceInStack  `json:"instances"`
	Pagination         PaginationInfo     `json:"pagination"`
}

type InstanceInStack struct {
	ID                 string              `json:"id"`
	DefinitionID       string              `json:"definition_id"`
	DefinitionName     string              `json:"definition_name"`
	Quantity           int                 `json:"quantity"`
	FieldValues        []InstanceFieldValue `json:"field_values"`
	LocationID         *string             `json:"location_id"`
	LocationName       *string             `json:"location_name"`
	ParentInstanceID   *string             `json:"parent_instance_id"`
	ParentInstanceName *string             `json:"parent_instance_name"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
}

type PaginationInfo struct {
	Page            int `json:"page"`
	PerPage         int `json:"per_page"`
	TotalPages      int `json:"total_pages"`
	TotalInstances  int `json:"total_instances"`
}

type MoveStackInput struct {
	DefinitionID           string  `json:"definition_id"`
	SourceLocationID       *string `json:"source_location_id"`
	SourceParentInstanceID *string `json:"source_parent_instance_id"`
	Quantity               int     `json:"quantity"`
	TargetLocationID       *string `json:"target_location_id"`
	TargetParentInstanceID *string `json:"target_parent_instance_id"`
}

type MoveStackResult struct {
	MovedQuantity int         `json:"moved_quantity"`
	Source        *StackDetail `json:"source"`
	Target        *StackDetail `json:"target"`
}

type StackService struct {
	db *sql.DB
}

func NewStackService(db *sql.DB) *StackService {
	return &StackService{db: db}
}

func stackListBaseQuery(whereClause string) string {
	return fmt.Sprintf(`
		SELECT
			d.id AS definition_id,
			d.name AS definition_name,
			d.unit,
			d.is_container,
			i.location_id,
			l.name AS location_name,
			i.parent_instance_id,
			pi_def.name AS parent_instance_name,
			COALESCE(SUM(i.quantity), 0) AS total_quantity,
			COUNT(i.id) AS instance_count
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		LEFT JOIN locations l ON l.id = i.location_id
		LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
		LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
		%s
		GROUP BY d.id, i.location_id, i.parent_instance_id
		ORDER BY d.name ASC
	`, whereClause)
}

func (s *StackService) List(locationID, parentInstanceID *string) (*StackListResult, error) {
	var whereClause string
	var args []interface{}

	if locationID != nil {
		whereClause = "WHERE i.location_id = ?"
		args = append(args, *locationID)
	} else if parentInstanceID != nil {
		whereClause = "WHERE i.parent_instance_id = ?"
		args = append(args, *parentInstanceID)
	}

	query := stackListBaseQuery(whereClause) + " LIMIT 501"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stacks: %w", err)
	}
	defer rows.Close()

	var stacks []BrowseStack
	for rows.Next() {
		var stack BrowseStack
		var locID, locName, parentInstID, parentInstName sql.NullString
		var unit sql.NullString

		if err := rows.Scan(
			&stack.DefinitionID, &stack.DefinitionName, &unit, &stack.IsContainer,
			&locID, &locName, &parentInstID, &parentInstName,
			&stack.TotalQuantity, &stack.InstanceCount,
		); err != nil {
			return nil, fmt.Errorf("scan stack: %w", err)
		}

		if unit.Valid {
			stack.Unit = &unit.String
		}
		if locID.Valid {
			stack.LocationID = &locID.String
			stack.LocationName = &locName.String
		}
		if parentInstID.Valid {
			stack.ParentInstanceID = &parentInstID.String
			stack.ParentInstanceName = &parentInstName.String
		}

		stacks = append(stacks, stack)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalCount := len(stacks)
	truncated := totalCount > 500
	if truncated {
		stacks = stacks[:500]
	}

	result := &StackListResult{
		Stacks:     stacks,
		TotalCount: totalCount,
		Truncated:  truncated,
	}

	if result.Stacks == nil {
		result.Stacks = []BrowseStack{}
	}

	return result, nil
}

func (s *StackService) GetDetail(definitionID string, locationID, parentInstanceID *string, page, perPage int) (*StackDetail, error) {
	if (locationID == nil) == (parentInstanceID == nil) {
		return nil, fmt.Errorf("exactly one of location_id or parent_instance_id must be provided: %w", ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	var totalCount int
	var whereField string
	var args []interface{}
	args = append(args, definitionID)

	if locationID != nil {
		whereField = "AND i.location_id = ?"
		args = append(args, *locationID)
	} else {
		whereField = "AND i.parent_instance_id = ?"
		args = append(args, *parentInstanceID)
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM item_instances i
		WHERE i.definition_id = ? %s
	`, whereField)
	if err := s.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("count stack instances: %w", err)
	}
	if totalCount == 0 {
		return nil, ErrNotFound
	}

	var defName string
	var isContainer bool
	var parentDefID, parentDefName, unit sql.NullString
	err := s.db.QueryRow(`
		SELECT d.name, d.is_container, d.unit, pd.id, pd.name
		FROM item_definitions d
		LEFT JOIN item_definitions pd ON pd.id = d.parent_def_id
		WHERE d.id = ?
	`, definitionID).Scan(&defName, &isContainer, &unit, &parentDefID, &parentDefName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("definition not found: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get definition: %w", err)
	}

	var totalQty int
	aggArgs := []interface{}{definitionID}
	if locationID != nil {
		aggArgs = append(aggArgs, *locationID)
		s.db.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0) FROM item_instances
			WHERE definition_id = ? AND location_id = ?
		`, definitionID, *locationID).Scan(&totalQty)
	} else {
		aggArgs = append(aggArgs, *parentInstanceID)
		s.db.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0) FROM item_instances
			WHERE definition_id = ? AND parent_instance_id = ?
		`, definitionID, *parentInstanceID).Scan(&totalQty)
	}

	var childCount int
	childQuery := `
		SELECT COALESCE(SUM(c.child_count), 0) FROM (
			SELECT COUNT(*) as child_count
			FROM item_instances child
			WHERE child.parent_instance_id IN (
				SELECT id FROM item_instances
				WHERE definition_id = ? %s
			)
			GROUP BY child.parent_instance_id
		) c
	`
	childQuery = fmt.Sprintf(childQuery, whereField)
	s.db.QueryRow(childQuery, args...).Scan(&childCount)

	var placementLocID, placementLocName, placementParentInstID, placementParentInstName *string

	if locationID != nil {
		placementLocID = locationID
		var locName string
		err := s.db.QueryRow(`SELECT name FROM locations WHERE id = ?`, *locationID).Scan(&locName)
		if err != nil {
			return nil, fmt.Errorf("get location name: %w", err)
		}
		placementLocName = &locName
	} else {
		placementParentInstID = parentInstanceID
		var piDefName string
		err := s.db.QueryRow(`
			SELECT d.name FROM item_definitions d
			JOIN item_instances i ON i.definition_id = d.id
			WHERE i.id = ?
		`, *parentInstanceID).Scan(&piDefName)
		if err != nil {
			return nil, fmt.Errorf("get parent instance name: %w", err)
		}
		placementParentInstName = &piDefName
	}

	breadcrumb, err := s.resolveStackBreadcrumb(definitionID, locationID, parentInstanceID)
	if err != nil {
		return nil, fmt.Errorf("resolve breadcrumb: %w", err)
	}

	offset := (page - 1) * perPage
	instQuery := fmt.Sprintf(`
		SELECT
			i.id, i.definition_id, d.name,
			i.quantity, i.location_id, l.name,
			i.parent_instance_id, pi_def.name,
			i.created_at, i.updated_at
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		LEFT JOIN locations l ON l.id = i.location_id
		LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
		LEFT JOIN item_definitions pi_def ON pi_def.id = pi.definition_id
		WHERE i.definition_id = ? %s
		ORDER BY i.updated_at DESC
		LIMIT ? OFFSET ?
	`, whereField)

	instArgs := []interface{}{definitionID}
	if locationID != nil {
		instArgs = append(instArgs, *locationID)
	} else {
		instArgs = append(instArgs, *parentInstanceID)
	}
	instArgs = append(instArgs, perPage, offset)

	instRows, err := s.db.Query(instQuery, instArgs...)
	if err != nil {
		return nil, fmt.Errorf("query stack instances: %w", err)
	}
	defer instRows.Close()

	var instances []InstanceInStack
	for instRows.Next() {
		var inst InstanceInStack
		var iLocID, iLocName, iParentInstID, iParentInstName sql.NullString
		var createdAt, updatedAt string

		if err := instRows.Scan(
			&inst.ID, &inst.DefinitionID, &inst.DefinitionName,
			&inst.Quantity, &iLocID, &iLocName,
			&iParentInstID, &iParentInstName,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan instance in stack: %w", err)
		}

		if iLocID.Valid {
			inst.LocationID = &iLocID.String
			inst.LocationName = &iLocName.String
		}
		if iParentInstID.Valid {
			inst.ParentInstanceID = &iParentInstID.String
			inst.ParentInstanceName = &iParentInstName.String
		}
		inst.CreatedAt = createdAt
		inst.UpdatedAt = updatedAt

		fvRows, err := s.db.Query(`
			SELECT ifv.field_id, df.field_name, df.field_type, df.enum_values, ifv.value
			FROM instance_field_values ifv
			JOIN definition_fields df ON df.id = ifv.field_id
			WHERE ifv.instance_id = ?
			ORDER BY df.display_order ASC
			LIMIT 5
		`, inst.ID)
		if err != nil {
			return nil, fmt.Errorf("query field values: %w", err)
		}
		var fieldValues []InstanceFieldValue
		for fvRows.Next() {
			var fv InstanceFieldValue
			var value sql.NullString
			var enumValues sql.NullString
			if err := fvRows.Scan(&fv.FieldID, &fv.FieldName, &fv.FieldType, &enumValues, &value); err != nil {
				fvRows.Close()
				return nil, fmt.Errorf("scan field value: %w", err)
			}
			if value.Valid {
				fv.Value = &value.String
			}
			if enumValues.Valid && enumValues.String != "" {
				raw := json.RawMessage(enumValues.String)
				fv.EnumValues = &raw
			}
			fieldValues = append(fieldValues, fv)
		}
		fvRows.Close()
		if err := fvRows.Err(); err != nil {
			return nil, err
		}

		if fieldValues == nil {
			fieldValues = []InstanceFieldValue{}
		}
		inst.FieldValues = fieldValues
		instances = append(instances, inst)
	}
	if err := instRows.Err(); err != nil {
		return nil, err
	}

	if instances == nil {
		instances = []InstanceInStack{}
	}

	totalPages := (totalCount + perPage - 1) / perPage

	detail := &StackDetail{
		DefinitionID:       definitionID,
		DefinitionName:     defName,
		Unit:               nil,
		IsContainer:        isContainer,
		ParentDefID:        nil,
		ParentDefName:      nil,
		LocationID:         placementLocID,
		LocationName:       placementLocName,
		ParentInstanceID:   placementParentInstID,
		ParentInstanceName: placementParentInstName,
		TotalQuantity:      totalQty,
		InstanceCount:      totalCount,
		ChildCount:         childCount,
		Breadcrumb:         breadcrumb,
		Instances:          instances,
		Pagination: PaginationInfo{
			Page:           page,
			PerPage:        perPage,
			TotalPages:     totalPages,
			TotalInstances: totalCount,
		},
	}

	if unit.Valid {
		detail.Unit = &unit.String
	}
	if parentDefID.Valid {
		detail.ParentDefID = &parentDefID.String
		detail.ParentDefName = &parentDefName.String
	}

	return detail, nil
}

func (s *StackService) resolveStackBreadcrumb(definitionID string, locationID, parentInstanceID *string) ([]BreadcrumbEntry, error) {
	if locationID != nil {
		rows, err := s.db.Query(`
			WITH RECURSIVE ancestors AS (
				SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
				UNION ALL
				SELECT l.id, l.name, l.parent_id, a.depth + 1
				FROM locations l JOIN ancestors a ON l.id = a.parent_id
			)
			SELECT id, name FROM ancestors ORDER BY depth DESC
		`, *locationID)
		if err != nil {
			return nil, fmt.Errorf("get location breadcrumb: %w", err)
		}
		defer rows.Close()

		var entries []BreadcrumbEntry
		for rows.Next() {
			var e BreadcrumbEntry
			if err := rows.Scan(&e.ID, &e.Name); err != nil {
				return nil, fmt.Errorf("scan breadcrumb: %w", err)
			}
			e.Kind = "location"
			entries = append(entries, e)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return entries, nil
	}

	rows, err := s.db.Query(`
		WITH RECURSIVE instance_chain AS (
			SELECT id, parent_instance_id, location_id, 0 AS depth
			FROM item_instances WHERE id = ?
			UNION ALL
			SELECT i.id, i.parent_instance_id, i.location_id, c.depth + 1
			FROM item_instances i JOIN instance_chain c ON i.id = c.parent_instance_id
		)
		SELECT id, parent_instance_id, location_id, depth FROM instance_chain ORDER BY depth ASC
	`, *parentInstanceID)
	if err != nil {
		return nil, fmt.Errorf("get instance chain: %w", err)
	}
	defer rows.Close()

	type chainEntry struct {
		id             string
		parentInstID   sql.NullString
		locationID     sql.NullString
		depth          int
	}
	var chain []chainEntry
	for rows.Next() {
		var ce chainEntry
		if err := rows.Scan(&ce.id, &ce.parentInstID, &ce.locationID, &ce.depth); err != nil {
			return nil, fmt.Errorf("scan chain: %w", err)
		}
		chain = append(chain, ce)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return nil, fmt.Errorf("instance chain is empty: %w", ErrNotFound)
	}

	var locationIDFromChain *string
	for _, ce := range chain {
		if ce.locationID.Valid {
			locID := ce.locationID.String
			locationIDFromChain = &locID
			break
		}
	}

	var entries []BreadcrumbEntry

	if locationIDFromChain != nil {
		locRows, err := s.db.Query(`
			WITH RECURSIVE ancestors AS (
				SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
				UNION ALL
				SELECT l.id, l.name, l.parent_id, a.depth + 1
				FROM locations l JOIN ancestors a ON l.id = a.parent_id
			)
			SELECT id, name FROM ancestors ORDER BY depth DESC
		`, *locationIDFromChain)
		if err != nil {
			return nil, fmt.Errorf("get location breadcrumb: %w", err)
		}
		for locRows.Next() {
			var e BreadcrumbEntry
			if err := locRows.Scan(&e.ID, &e.Name); err != nil {
				locRows.Close()
				return nil, fmt.Errorf("scan loc breadcrumb: %w", err)
			}
			e.Kind = "location"
			entries = append(entries, e)
		}
		locRows.Close()
		if err := locRows.Err(); err != nil {
			return nil, err
		}
	}

	for i := len(chain) - 1; i >= 0; i-- {
		var instName string
		err := s.db.QueryRow(`
			SELECT d.name FROM item_definitions d
			JOIN item_instances i ON i.definition_id = d.id
			WHERE i.id = ?
		`, chain[i].id).Scan(&instName)
		if err != nil {
			return nil, fmt.Errorf("get instance name: %w", err)
		}
		entries = append(entries, BreadcrumbEntry{
			ID:   chain[i].id,
			Name: instName,
			Kind: "instance",
		})
	}

	return entries, nil
}

func (s *StackService) Move(input MoveStackInput) (*MoveStackResult, error) {
	if strings.TrimSpace(input.DefinitionID) == "" {
		return nil, fmt.Errorf("definition_id is required: %w", ErrInvalidInput)
	}
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0: %w", ErrInvalidInput)
	}
	if (input.SourceLocationID == nil) == (input.SourceParentInstanceID == nil) {
		return nil, fmt.Errorf("exactly one of source_location_id or source_parent_instance_id must be provided: %w", ErrInvalidInput)
	}
	if (input.TargetLocationID == nil) == (input.TargetParentInstanceID == nil) {
		return nil, fmt.Errorf("exactly one of target_location_id or target_parent_instance_id must be provided: %w", ErrInvalidInput)
	}

	var totalAvailable int
	if input.SourceLocationID != nil {
		s.db.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0) FROM item_instances
			WHERE definition_id = ? AND location_id = ?
		`, input.DefinitionID, *input.SourceLocationID).Scan(&totalAvailable)
	} else {
		s.db.QueryRow(`
			SELECT COALESCE(SUM(quantity), 0) FROM item_instances
			WHERE definition_id = ? AND parent_instance_id = ?
		`, input.DefinitionID, *input.SourceParentInstanceID).Scan(&totalAvailable)
	}

	if input.Quantity > totalAvailable {
		return nil, fmt.Errorf("Cannot move %d items: only %d available.", input.Quantity, totalAvailable)
	}

	if input.TargetParentInstanceID != nil {
		var tgtExists bool
		s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_instances WHERE id = ?)`, *input.TargetParentInstanceID).Scan(&tgtExists)
		if !tgtExists {
			return nil, fmt.Errorf("target parent instance not found: %w", ErrNotFound)
		}
		var tgtDefIsContainer bool
		s.db.QueryRow(`
			SELECT d.is_container FROM item_definitions d
			JOIN item_instances i ON i.definition_id = d.id
			WHERE i.id = ?
		`, *input.TargetParentInstanceID).Scan(&tgtDefIsContainer)
		if !tgtDefIsContainer {
			return nil, fmt.Errorf("target parent instance is not a container: %w", ErrInvalidInput)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var sourceWhere string
	var sourceArgs []interface{}
	sourceArgs = append(sourceArgs, input.DefinitionID)
	if input.SourceLocationID != nil {
		sourceWhere = "AND i.location_id = ?"
		sourceArgs = append(sourceArgs, *input.SourceLocationID)
	} else {
		sourceWhere = "AND i.parent_instance_id = ?"
		sourceArgs = append(sourceArgs, *input.SourceParentInstanceID)
	}

	instQuery := fmt.Sprintf(`
		SELECT i.id, i.quantity, i.definition_id, i.location_id, i.parent_instance_id
		FROM item_instances i
		WHERE i.definition_id = ? %s
		ORDER BY i.created_at ASC
	`, sourceWhere)

	instRows, err := tx.Query(instQuery, sourceArgs...)
	if err != nil {
		return nil, fmt.Errorf("query source instances: %w", err)
	}

	type sourceInst struct {
		id             string
		quantity       int
		definitionID   string
		locationID     sql.NullString
		parentInstID   sql.NullString
	}
	var sourceInstances []sourceInst
	for instRows.Next() {
		var si sourceInst
		if err := instRows.Scan(&si.id, &si.quantity, &si.definitionID, &si.locationID, &si.parentInstID); err != nil {
			instRows.Close()
			return nil, fmt.Errorf("scan source instance: %w", err)
		}
		sourceInstances = append(sourceInstances, si)
	}
	instRows.Close()
	if err := instRows.Err(); err != nil {
		return nil, err
	}

	var movedInstanceIDs []string
	remaining := input.Quantity

	for _, si := range sourceInstances {
		if remaining <= 0 {
			break
		}

		var take int
		if si.quantity <= remaining {
			take = si.quantity
		} else {
			take = remaining
		}

		hasChildren := false
		if take == si.quantity {
			var childCnt int
			err := tx.QueryRow(`SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = ?`, si.id).Scan(&childCnt)
			if err != nil {
				return nil, fmt.Errorf("check children: %w", err)
			}
			if childCnt > 0 {
				continue
			}
		}

		var targetLocID interface{}
		var targetParentInstID interface{}
		if input.TargetLocationID != nil {
			targetLocID = *input.TargetLocationID
			targetParentInstID = nil
		} else {
			targetLocID = nil
			targetParentInstID = *input.TargetParentInstanceID
		}

		var newQuantity int
		if take < si.quantity {
			newQuantity = si.quantity - take
		} else {
			newQuantity = 0
		}

		if newQuantity == 0 {
			_, err := tx.Exec(`DELETE FROM item_instances WHERE id = ?`, si.id)
			if err != nil {
				return nil, fmt.Errorf("delete exhausted source: %w", err)
			}
		} else {
			_, err := tx.Exec(
				`UPDATE item_instances SET quantity = quantity - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				take, si.id,
			)
			if err != nil {
				return nil, fmt.Errorf("decrement source: %w", err)
			}
		}

		matchingID := s.findMatchingStackTargetTx(tx, si.definitionID, input.TargetLocationID, input.TargetParentInstanceID)
		if matchingID != nil {
			_, err := tx.Exec(
				`UPDATE item_instances SET quantity = quantity + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				take, *matchingID,
			)
			if err != nil {
				return nil, fmt.Errorf("merge at target: %w", err)
			}
		} else {
			targetID := uuid.New().String()
			_, err = tx.Exec(
				`INSERT INTO item_instances (id, definition_id, quantity, location_id, parent_instance_id) VALUES (?, ?, ?, ?, ?)`,
				targetID, si.definitionID, take, targetLocID, targetParentInstID,
			)
			if err != nil {
				return nil, fmt.Errorf("create target instance: %w", err)
			}

			fvRows, err := s.db.Query(`
				SELECT field_id, value FROM instance_field_values WHERE instance_id = ?
			`, si.id)
			if err == nil {
				for fvRows.Next() {
					var fieldID, value string
					var val sql.NullString
					if err := fvRows.Scan(&fieldID, &val); err != nil {
						fvRows.Close()
						return nil, fmt.Errorf("scan src field value: %w", err)
					}
					if val.Valid {
						value = val.String
					}
					fvID := uuid.New().String()
					_, err := tx.Exec(
						`INSERT INTO instance_field_values (id, instance_id, field_id, value) VALUES (?, ?, ?, ?)`,
						fvID, targetID, fieldID, val,
					)
					if err != nil {
						fvRows.Close()
						return nil, fmt.Errorf("insert target field value: %w", err)
					}
					_ = value
				}
				fvRows.Close()
			}
		}

		movedInstanceIDs = append(movedInstanceIDs, si.id)
		remaining -= take
		_ = hasChildren
	}

	if remaining > 0 {
		return nil, fmt.Errorf("Cannot move all items: some instances have children or insufficient quantity.")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	var sourceLocID, sourceParentInstID *string
	if input.SourceLocationID != nil {
		sourceLocID = input.SourceLocationID
	} else {
		sourceParentInstID = input.SourceParentInstanceID
	}
	sourceDetail, _ := s.GetDetail(input.DefinitionID, sourceLocID, sourceParentInstID, 1, 50)

	var targetLocID, targetParentInstID *string
	if input.TargetLocationID != nil {
		targetLocID = input.TargetLocationID
	} else {
		targetParentInstID = input.TargetParentInstanceID
	}
	targetDetail, err := s.GetDetail(input.DefinitionID, targetLocID, targetParentInstID, 1, 50)
	if err != nil {
		emptyDetail := &StackDetail{}
		return &MoveStackResult{
			MovedQuantity: input.Quantity - remaining,
			Source:        sourceDetail,
			Target:        emptyDetail,
		}, nil
	}

	return &MoveStackResult{
		MovedQuantity: input.Quantity - remaining,
		Source:        sourceDetail,
		Target:        targetDetail,
	}, nil
}

func (s *StackService) findMatchingStackTargetTx(tx *sql.Tx, definitionID string, targetLocationID, targetParentInstanceID *string) *string {
	var where string
	var args []interface{}
	args = append(args, definitionID)

	if targetLocationID != nil {
		where = "AND i.location_id = ?"
		args = append(args, *targetLocationID)
	} else {
		where = "AND i.parent_instance_id = ?"
		args = append(args, *targetParentInstanceID)
	}

	query := fmt.Sprintf(`
		SELECT i.id FROM item_instances i
		WHERE i.definition_id = ? %s
		LIMIT 1
	`, where)

	var matchingID sql.NullString
	err := tx.QueryRow(query, args...).Scan(&matchingID)
	if err != nil {
		return nil
	}
	if matchingID.Valid {
		return &matchingID.String
	}
	return nil
}

func (s *StackService) Delete(definitionID string, locationID, parentInstanceID *string) error {
	if (locationID == nil) == (parentInstanceID == nil) {
		return fmt.Errorf("exactly one of location_id or parent_instance_id must be provided: %w", ErrInvalidInput)
	}

	var where string
	var args []interface{}
	args = append(args, definitionID)

	if locationID != nil {
		where = "AND location_id = ?"
		args = append(args, *locationID)
	} else {
		where = "AND parent_instance_id = ?"
		args = append(args, *parentInstanceID)
	}

	idQuery := fmt.Sprintf(`SELECT id FROM item_instances WHERE definition_id = ? %s`, where)
	rows, err := s.db.Query(idQuery, args...)
	if err != nil {
		return fmt.Errorf("query stack instance ids: %w", err)
	}

	var instanceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan instance id: %w", err)
		}
		instanceIDs = append(instanceIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if len(instanceIDs) == 0 {
		return ErrNotFound
	}

	var childCount int
	childArgs := make([]interface{}, len(instanceIDs))
	childPlaceholders := make([]string, len(instanceIDs))
	for i, id := range instanceIDs {
		childPlaceholders[i] = "?"
		childArgs[i] = id
	}
	childQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM item_instances
		WHERE parent_instance_id IN (%s)
	`, strings.Join(childPlaceholders, ","))
	s.db.QueryRow(childQuery, childArgs...).Scan(&childCount)
	if childCount > 0 {
		return fmt.Errorf("Cannot delete stack: %d instances have items stored inside them.", childCount)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	deleteQuery := fmt.Sprintf(`DELETE FROM item_instances WHERE definition_id = ? %s`, where)
	_, err = tx.Exec(deleteQuery, args...)
	if err != nil {
		return fmt.Errorf("delete stack instances: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
