package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type DefinitionSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	ParentDefID     *string `json:"parent_def_id"`
	ParentDefName   *string `json:"parent_def_name"`
	Unit            *string `json:"unit"`
	IsContainer     bool   `json:"is_container"`
	TotalInstances  int    `json:"total_instances"`
	Tags            []Tag  `json:"tags"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type DefinitionField struct {
	ID                string  `json:"id"`
	FieldName         string  `json:"field_name"`
	FieldType         string  `json:"field_type"`
	EnumValues        *json.RawMessage `json:"enum_values"`
	IsRequired        bool    `json:"is_required"`
	DisplayOrder      int     `json:"display_order"`
	DefaultValue      *string `json:"default_value"`
	IsChildEditable   bool    `json:"is_child_editable"`
	InheritedFromDefID *string `json:"inherited_from_def_id"`
}

type InstanceSummaryDetail struct {
	TotalInstances int                      `json:"total_instances"`
	TotalQuantity  int                      `json:"total_quantity"`
	ByLocation     []LocationInstanceCount  `json:"by_location"`
	ByParentInstance []ParentInstanceCount  `json:"by_parent_instance"`
}

type LocationInstanceCount struct {
	LocationID    string `json:"location_id"`
	LocationName  string `json:"location_name"`
	InstanceCount int    `json:"instance_count"`
	TotalQuantity int    `json:"total_quantity"`
}

type ParentInstanceCount struct {
	ParentInstanceID   string `json:"parent_instance_id"`
	ParentInstanceName string `json:"parent_instance_name"`
	LocationID         string `json:"location_id"`
	LocationName       string `json:"location_name"`
	InstanceCount      int    `json:"instance_count"`
	TotalQuantity      int    `json:"total_quantity"`
}

type DefinitionDetail struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	Description         *string               `json:"description"`
	ParentDefID         *string               `json:"parent_def_id"`
	ParentDefName       *string               `json:"parent_def_name"`
	Unit                *string               `json:"unit"`
	IsContainer         bool                  `json:"is_container"`
	CreatedAt           string                `json:"created_at"`
	UpdatedAt           string                `json:"updated_at"`
	Fields              []DefinitionField     `json:"fields"`
	Tags                []Tag                 `json:"tags"`
	InstancesSummary    InstanceSummaryDetail `json:"instances_summary"`
	ChildDefinitionCount int                  `json:"child_definition_count"`
}

type CreateDefinitionInput struct {
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	ParentDefID *string           `json:"parent_def_id"`
	Unit        *string           `json:"unit"`
	IsContainer *bool             `json:"is_container"`
	Fields      []CreateFieldInput `json:"fields"`
	TagIDs      []string          `json:"tag_ids"`
}

type CreateFieldInput struct {
	FieldName       string          `json:"field_name"`
	FieldType       string          `json:"field_type"`
	EnumValues      *json.RawMessage `json:"enum_values"`
	IsRequired      bool            `json:"is_required"`
	DisplayOrder    int             `json:"display_order"`
	DefaultValue    *string         `json:"default_value"`
	IsChildEditable bool            `json:"is_child_editable"`
}

type UpdateDefinitionInput struct {
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	ParentDefID *string           `json:"parent_def_id"`
	Unit        *string           `json:"unit"`
	IsContainer *bool             `json:"is_container"`
	Fields      *[]CreateFieldInput `json:"fields"`
	TagIDs      *[]string         `json:"tag_ids"`
}

type OverrideInput struct {
	ParentFieldID string  `json:"parent_field_id"`
	DefaultValue  *string `json:"default_value"`
}

type Override struct {
	DefinitionID   string  `json:"definition_id"`
	ParentFieldID  string  `json:"parent_field_id"`
	DefaultValue   *string `json:"default_value"`
}

type DefinitionService struct {
	db *sql.DB
}

func NewDefinitionService(db *sql.DB) *DefinitionService {
	return &DefinitionService{db: db}
}

func (s *DefinitionService) GetAll() ([]DefinitionSummary, error) {
	rows, err := s.db.Query(`
		SELECT d.id, d.name, COALESCE(d.description, ''), d.parent_def_id, d.unit, d.is_container,
		       d.created_at, d.updated_at
		FROM item_definitions d
		ORDER BY d.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list definitions: %w", err)
	}
	defer rows.Close()

	var defs []DefinitionSummary
	for rows.Next() {
		var d DefinitionSummary
		var desc string
		if err := rows.Scan(&d.ID, &d.Name, &desc, &d.ParentDefID, &d.Unit, &d.IsContainer, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan definition: %w", err)
		}
		if desc != "" {
			d.Description = desc
		}
		defs = append(defs, d)
	}
	if defs == nil {
		return []DefinitionSummary{}, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range defs {
		if defs[i].ParentDefID != nil {
			var parentName string
			err := s.db.QueryRow(`SELECT name FROM item_definitions WHERE id = ?`, *defs[i].ParentDefID).Scan(&parentName)
			if err == nil {
				defs[i].ParentDefName = &parentName
			}
		}

		defs[i].Tags, _ = s.getDefinitionTags(defs[i].ID)

		var totalInstances int
		s.db.QueryRow(`SELECT COUNT(*) FROM item_instances WHERE definition_id = ?`, defs[i].ID).Scan(&totalInstances)
		defs[i].TotalInstances = totalInstances
	}

	return defs, nil
}

func (s *DefinitionService) GetByID(id string) (*DefinitionDetail, error) {
	base, err := s.getBaseDefinition(id)
	if err != nil {
		return nil, err
	}

	fields, err := s.ResolveFields(id)
	if err != nil {
		return nil, err
	}

	tags, err := s.getDefinitionTags(id)
	if err != nil {
		return nil, err
	}

	summary, err := s.getInstanceSummary(id)
	if err != nil {
		return nil, err
	}

	var childCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM item_definitions WHERE parent_def_id = ?`, id).Scan(&childCount)

	return &DefinitionDetail{
		ID:                  base.ID,
		Name:                base.Name,
		Description:         base.Description,
		ParentDefID:         base.ParentDefID,
		ParentDefName:       base.ParentDefName,
		Unit:                base.Unit,
		IsContainer:         base.IsContainer,
		CreatedAt:           base.CreatedAt,
		UpdatedAt:           base.UpdatedAt,
		Fields:              fields,
		Tags:                tags,
		InstancesSummary:    summary,
		ChildDefinitionCount: childCount,
	}, nil
}

func (s *DefinitionService) getBaseDefinition(id string) (*DefinitionDetail, error) {
	var d DefinitionDetail
	var desc sql.NullString
	err := s.db.QueryRow(`
		SELECT id, name, description, parent_def_id, unit, is_container, created_at, updated_at
		FROM item_definitions WHERE id = ?
	`, id).Scan(&d.ID, &d.Name, &desc, &d.ParentDefID, &d.Unit, &d.IsContainer, &d.CreatedAt, &d.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get definition: %w", err)
	}

	if desc.Valid {
		d.Description = &desc.String
	}

	if d.ParentDefID != nil {
		var parentName string
		err := s.db.QueryRow(`SELECT name FROM item_definitions WHERE id = ?`, *d.ParentDefID).Scan(&parentName)
		if err == nil {
			d.ParentDefName = &parentName
		}
	}

	return &d, nil
}

func (s *DefinitionService) getAncestorIDs(defID string) ([]string, error) {
	var ancestors []string
	currentID := defID

	for {
		var parentID sql.NullString
		err := s.db.QueryRow(`SELECT parent_def_id FROM item_definitions WHERE id = ?`, currentID).Scan(&parentID)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, fmt.Errorf("get parent: %w", err)
		}
		if !parentID.Valid {
			break
		}
		ancestors = append(ancestors, parentID.String)
		currentID = parentID.String
	}

	return ancestors, nil
}

func (s *DefinitionService) getAncestorIDsUntil(defID, stopID string) ([]string, error) {
	var ancestors []string
	currentID := defID

	for {
		var parentID sql.NullString
		err := s.db.QueryRow(`SELECT parent_def_id FROM item_definitions WHERE id = ?`, currentID).Scan(&parentID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("definition not found in chain: %w", ErrInvalidInput)
			}
			return nil, fmt.Errorf("get parent: %w", err)
		}
		if !parentID.Valid {
			return nil, fmt.Errorf("not a descendant: %w", ErrInvalidInput)
		}
		ancestors = append(ancestors, parentID.String)
		if parentID.String == stopID {
			return ancestors, nil
		}
		currentID = parentID.String
	}
}

func (s *DefinitionService) isDescendant(defID, potentialAncestor string) (bool, error) {
	var parentID sql.NullString
	currentID := defID

	for {
		err := s.db.QueryRow(`SELECT parent_def_id FROM item_definitions WHERE id = ?`, currentID).Scan(&parentID)
		if err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, fmt.Errorf("check descendant: %w", err)
		}
		if !parentID.Valid {
			return false, nil
		}
		if parentID.String == potentialAncestor {
			return true, nil
		}
		currentID = parentID.String
	}
}

func (s *DefinitionService) ResolveFields(defID string) ([]DefinitionField, error) {
	ancestors, err := s.getAncestorIDs(defID)
	if err != nil {
		return nil, err
	}

	ownFields, err := s.getOwnFields(defID)
	if err != nil {
		return nil, err
	}

	var resolved []DefinitionField

	for _, f := range ownFields {
		f.InheritedFromDefID = nil
		resolved = append(resolved, f)
	}

	for _, ancestorID := range ancestors {
		afields, err := s.getOwnFields(ancestorID)
		if err != nil {
			return nil, err
		}
		for _, af := range afields {
			inhFrom := ancestorID
			af.InheritedFromDefID = &inhFrom

			overrideVal := s.getOverride(defID, af.ID)
			if overrideVal != nil {
				af.DefaultValue = overrideVal
			}

			resolved = append(resolved, af)
		}
	}

	sort.SliceStable(resolved, func(i, j int) bool {
		return resolved[i].DisplayOrder < resolved[j].DisplayOrder
	})

	if resolved == nil {
		resolved = []DefinitionField{}
	}

	return resolved, nil
}

func (s *DefinitionService) getOwnFields(defID string) ([]DefinitionField, error) {
	rows, err := s.db.Query(`
		SELECT id, field_name, field_type, enum_values, is_required, display_order,
		       default_value, is_child_editable
		FROM definition_fields
		WHERE definition_id = ?
		ORDER BY display_order ASC
	`, defID)
	if err != nil {
		return nil, fmt.Errorf("get own fields: %w", err)
	}
	defer rows.Close()

	var fields []DefinitionField
	for rows.Next() {
		var f DefinitionField
		var enumRaw sql.NullString
		if err := rows.Scan(&f.ID, &f.FieldName, &f.FieldType, &enumRaw, &f.IsRequired,
			&f.DisplayOrder, &f.DefaultValue, &f.IsChildEditable); err != nil {
			return nil, fmt.Errorf("scan field: %w", err)
		}
		if enumRaw.Valid {
			raw := json.RawMessage(enumRaw.String)
			f.EnumValues = &raw
		}
		fields = append(fields, f)
	}

	if fields == nil {
		return []DefinitionField{}, nil
	}

	return fields, rows.Err()
}

func (s *DefinitionService) getOverride(defID, parentFieldID string) *string {
	var val sql.NullString
	err := s.db.QueryRow(
		`SELECT default_value FROM definition_field_overrides WHERE definition_id = ? AND parent_field_id = ?`,
		defID, parentFieldID,
	).Scan(&val)
	if err != nil {
		return nil
	}
	if val.Valid {
		return &val.String
	}
	return nil
}

func (s *DefinitionService) Create(input CreateDefinitionInput) (*DefinitionDetail, error) {
	name := strings.TrimSpace(input.Name)
	if len(name) < 2 || len(name) > 200 {
		return nil, fmt.Errorf("name must be between 2 and 200 characters: %w", ErrInvalidInput)
	}

	var desc *string
	if input.Description != nil {
		trimmed := strings.TrimSpace(*input.Description)
		if len(trimmed) > 2000 {
			return nil, fmt.Errorf("description must be at most 2000 characters: %w", ErrInvalidInput)
		}
		if trimmed != "" {
			desc = &trimmed
		}
	}

	var unit *string
	if input.Unit != nil {
		trimmed := strings.TrimSpace(*input.Unit)
		if len(trimmed) > 20 {
			return nil, fmt.Errorf("unit must be at most 20 characters: %w", ErrInvalidInput)
		}
		if trimmed != "" {
			unit = &trimmed
		}
	}

	if input.ParentDefID != nil {
		if _, err := s.getBaseDefinition(*input.ParentDefID); err != nil {
			return nil, fmt.Errorf("parent definition not found: %w", ErrInvalidInput)
		}
	}

	if err := s.validateFields(input.Fields); err != nil {
		return nil, err
	}

	isContainer := false
	if input.IsContainer != nil {
		isContainer = *input.IsContainer
	}

	id := uuid.New().String()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO item_definitions (id, name, description, parent_def_id, unit, is_container) VALUES (?, ?, ?, ?, ?, ?)`,
		id, name, desc, input.ParentDefID, unit, isContainer,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("Definition '%s' already exists", name)
		}
		return nil, fmt.Errorf("create definition: %w", err)
	}

	if err := s.insertFields(tx, id, input.Fields); err != nil {
		return nil, err
	}

	if err := s.setTags(tx, id, input.TagIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetByID(id)
}

func (s *DefinitionService) Update(id string, input UpdateDefinitionInput) (*DefinitionDetail, error) {
	existing, err := s.getBaseDefinition(id)
	if err != nil {
		return nil, err
	}

	var finalName string
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if len(trimmed) < 2 || len(trimmed) > 200 {
			return nil, fmt.Errorf("name must be between 2 and 200 characters: %w", ErrInvalidInput)
		}
		finalName = trimmed
	} else {
		finalName = existing.Name
	}

	var finalDesc *string
	if input.Description != nil {
		trimmed := strings.TrimSpace(*input.Description)
		if len(trimmed) > 2000 {
			return nil, fmt.Errorf("description must be at most 2000 characters: %w", ErrInvalidInput)
		}
		if trimmed != "" {
			finalDesc = &trimmed
		}
	} else {
		finalDesc = existing.Description
	}

	var finalUnit *string
	if input.Unit != nil {
		trimmed := strings.TrimSpace(*input.Unit)
		if len(trimmed) > 20 {
			return nil, fmt.Errorf("unit must be at most 20 characters: %w", ErrInvalidInput)
		}
		if trimmed != "" {
			finalUnit = &trimmed
		}
	} else {
		finalUnit = existing.Unit
	}

	finalIsContainer := existing.IsContainer
	if input.IsContainer != nil {
		finalIsContainer = *input.IsContainer
	}

	finalParentID := existing.ParentDefID
	if input.ParentDefID != nil {
		newParentID := *input.ParentDefID

		if newParentID == id {
			return nil, fmt.Errorf("cannot set parent to self: %w", ErrInvalidInput)
		}

		isDesc, err := s.isDescendant(newParentID, id)
		if err != nil {
			return nil, err
		}
		if isDesc {
			return nil, fmt.Errorf("cycle detected: %w", ErrInvalidInput)
		}

		if _, err := s.getBaseDefinition(newParentID); err != nil {
			if err == ErrNotFound {
				return nil, fmt.Errorf("parent definition not found: %w", ErrInvalidInput)
			}
			return nil, err
		}

		finalParentID = &newParentID
	}

	if input.Fields != nil {
		if err := s.validateFields(*input.Fields); err != nil {
			return nil, err
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	descVal := interface{}(nil)
	if finalDesc != nil {
		descVal = *finalDesc
	}

	_, err = tx.Exec(
		`UPDATE item_definitions SET name = ?, description = ?, parent_def_id = ?, unit = ?, is_container = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		finalName, descVal, finalParentID, finalUnit, finalIsContainer, id,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, fmt.Errorf("Definition '%s' already exists", finalName)
		}
		return nil, fmt.Errorf("update definition: %w", err)
	}

	if input.Fields != nil {
		if _, err := tx.Exec(`DELETE FROM definition_fields WHERE definition_id = ?`, id); err != nil {
			return nil, fmt.Errorf("delete existing fields: %w", err)
		}

		if err := s.insertFields(tx, id, *input.Fields); err != nil {
			return nil, err
		}
	}

	if input.TagIDs != nil {
		if err := s.setTags(tx, id, *input.TagIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetByID(id)
}

func (s *DefinitionService) Delete(id string) error {
	var childCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM item_definitions WHERE parent_def_id = ?`, id).Scan(&childCount)

	var instanceCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM item_instances WHERE definition_id = ?`, id).Scan(&instanceCount)

	if childCount > 0 {
		return fmt.Errorf("Cannot delete: %d child definitions inherit from this definition", childCount)
	}

	if instanceCount > 0 {
		return fmt.Errorf("Cannot delete: definition has %d item instances", instanceCount)
	}

	result, err := s.db.Exec(`DELETE FROM item_definitions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete definition: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *DefinitionService) UpdateOverrides(defID string, overrides []OverrideInput) ([]Override, error) {
	if _, err := s.getBaseDefinition(defID); err != nil {
		return nil, err
	}

	for _, o := range overrides {
		var fieldDefID string
		var fieldType string
		var enumRaw sql.NullString
		var isChildEditable bool
		err := s.db.QueryRow(
			`SELECT df.definition_id, df.field_type, df.enum_values, df.is_child_editable
			 FROM definition_fields df WHERE df.id = ?`,
			o.ParentFieldID,
		).Scan(&fieldDefID, &fieldType, &enumRaw, &isChildEditable)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("parent field not found: %w", ErrInvalidInput)
			}
			return nil, fmt.Errorf("get parent field: %w", err)
		}

		isAncestorField := false
		{
			ancestors, err := s.getAncestorIDs(defID)
			if err != nil {
				return nil, err
			}
			for _, aID := range ancestors {
				if aID == fieldDefID {
					isAncestorField = true
					break
				}
			}
		}

		if !isAncestorField {
			return nil, fmt.Errorf("field does not belong to an ancestor definition: %w", ErrInvalidInput)
		}

		if !isChildEditable {
			return nil, fmt.Errorf("field is sealed and cannot be overridden: %w", ErrInvalidInput)
		}

		if o.DefaultValue != nil {
			if err := validateFieldDefaultValue(fieldType, o.DefaultValue, enumRaw); err != nil {
				return nil, err
			}
		}

		if o.DefaultValue != nil {
			_, err := s.db.Exec(
				`INSERT INTO definition_field_overrides (definition_id, parent_field_id, default_value)
				 VALUES (?, ?, ?)
				 ON CONFLICT(definition_id, parent_field_id) DO UPDATE SET default_value = excluded.default_value`,
				defID, o.ParentFieldID, *o.DefaultValue,
			)
			if err != nil {
				return nil, fmt.Errorf("upsert override: %w", err)
			}
		} else {
			_, err := s.db.Exec(
				`DELETE FROM definition_field_overrides WHERE definition_id = ? AND parent_field_id = ?`,
				defID, o.ParentFieldID,
			)
			if err != nil {
				return nil, fmt.Errorf("delete override: %w", err)
			}
		}
	}

	return s.getOverrides(defID)
}

func (s *DefinitionService) getOverrides(defID string) ([]Override, error) {
	rows, err := s.db.Query(
		`SELECT definition_id, parent_field_id, default_value
		 FROM definition_field_overrides WHERE definition_id = ?`,
		defID,
	)
	if err != nil {
		return nil, fmt.Errorf("get overrides: %w", err)
	}
	defer rows.Close()

	var overrides []Override
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.DefinitionID, &o.ParentFieldID, &o.DefaultValue); err != nil {
			return nil, fmt.Errorf("scan override: %w", err)
		}
		overrides = append(overrides, o)
	}

	if overrides == nil {
		overrides = []Override{}
	}

	return overrides, rows.Err()
}

func (s *DefinitionService) getDefinitionTags(defID string) ([]Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.color, t.created_at, t.updated_at
		FROM tags t
		JOIN definition_tags dt ON dt.tag_id = t.id
		WHERE dt.definition_id = ?
		ORDER BY t.name ASC
	`, defID)
	if err != nil {
		return nil, fmt.Errorf("get definition tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if tags == nil {
		tags = []Tag{}
	}

	return tags, rows.Err()
}

func (s *DefinitionService) insertFields(tx *sql.Tx, defID string, fields []CreateFieldInput) error {
	for _, f := range fields {
		fieldID := uuid.New().String()
		var enumValues interface{}
		if f.EnumValues != nil {
			rawJSON, err := json.Marshal(f.EnumValues)
			if err != nil {
				return fmt.Errorf("marshal enum_values: %w", err)
			}
			enumValues = string(rawJSON)
		}

		_, err := tx.Exec(
			`INSERT INTO definition_fields (id, definition_id, field_name, field_type, enum_values, is_required, display_order, default_value, is_child_editable)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fieldID, defID, f.FieldName, f.FieldType, enumValues, f.IsRequired, f.DisplayOrder, f.DefaultValue, f.IsChildEditable,
		)
		if err != nil {
			return fmt.Errorf("insert field: %w", err)
		}
	}
	return nil
}

func (s *DefinitionService) setTags(tx *sql.Tx, defID string, tagIDs []string) error {
	if _, err := tx.Exec(`DELETE FROM definition_tags WHERE definition_id = ?`, defID); err != nil {
		return fmt.Errorf("clear tags: %w", err)
	}

	for _, tagID := range tagIDs {
		var exists bool
		err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM tags WHERE id = ?)`, tagID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("tag not found: %w", ErrInvalidInput)
		}

		_, err = tx.Exec(
			`INSERT INTO definition_tags (definition_id, tag_id) VALUES (?, ?)`,
			defID, tagID,
		)
		if err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
	}
	return nil
}

func (s *DefinitionService) getInstanceSummary(defID string) (InstanceSummaryDetail, error) {
	var totalInstances int
	var totalQuantity int
	s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(quantity), 0) FROM item_instances WHERE definition_id = ?`,
		defID,
	).Scan(&totalInstances, &totalQuantity)

	var summaries InstanceSummaryDetail
	summaries.TotalInstances = totalInstances
	summaries.TotalQuantity = totalQuantity

	rows, err := s.db.Query(`
		SELECT i.location_id, l.name, COUNT(*), COALESCE(SUM(i.quantity), 0)
		FROM item_instances i
		JOIN locations l ON l.id = i.location_id
		WHERE i.definition_id = ? AND i.location_id IS NOT NULL
		GROUP BY i.location_id
		ORDER BY l.name ASC
	`, defID)
	if err != nil {
		return summaries, fmt.Errorf("get instance by location: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lic LocationInstanceCount
		if err := rows.Scan(&lic.LocationID, &lic.LocationName, &lic.InstanceCount, &lic.TotalQuantity); err != nil {
			return summaries, fmt.Errorf("scan location instance count: %w", err)
		}
		summaries.ByLocation = append(summaries.ByLocation, lic)
	}

	if summaries.ByLocation == nil {
		summaries.ByLocation = []LocationInstanceCount{}
	}

	if err := rows.Err(); err != nil {
		return summaries, err
	}

	parentRows, err := s.db.Query(`
		SELECT i.parent_instance_id, COUNT(*), COALESCE(SUM(i.quantity), 0)
		FROM item_instances i
		WHERE i.definition_id = ? AND i.parent_instance_id IS NOT NULL
		GROUP BY i.parent_instance_id
	`, defID)
	if err != nil {
		return summaries, fmt.Errorf("get instance by parent: %w", err)
	}
	defer parentRows.Close()

	for parentRows.Next() {
		var pic ParentInstanceCount
		if err := parentRows.Scan(&pic.ParentInstanceID, &pic.InstanceCount, &pic.TotalQuantity); err != nil {
			return summaries, fmt.Errorf("scan parent instance count: %w", err)
		}

		var parentDefName string
		s.db.QueryRow(`
			SELECT d.name FROM item_definitions d
			JOIN item_instances pi ON pi.definition_id = d.id
			WHERE pi.id = ?
		`, pic.ParentInstanceID).Scan(&parentDefName)
		pic.ParentInstanceName = parentDefName

		locID, locName := s.resolveLocationForInstance(pic.ParentInstanceID)
		pic.LocationID = locID
		pic.LocationName = locName

		summaries.ByParentInstance = append(summaries.ByParentInstance, pic)
	}

	if summaries.ByParentInstance == nil {
		summaries.ByParentInstance = []ParentInstanceCount{}
	}

	if err := parentRows.Err(); err != nil {
		return summaries, err
	}

	return summaries, nil
}

func (s *DefinitionService) validateFields(fields []CreateFieldInput) error {
	seenNames := make(map[string]bool)
	for _, f := range fields {
		name := strings.TrimSpace(f.FieldName)
		if len(name) < 1 || len(name) > 100 {
			return fmt.Errorf("field_name must be between 1 and 100 characters: %w", ErrInvalidInput)
		}

		validTypes := map[string]bool{"text": true, "number": true, "boolean": true, "date": true, "enum": true}
		if !validTypes[f.FieldType] {
			return fmt.Errorf("invalid field_type '%s': %w", f.FieldType, ErrInvalidInput)
		}

		if f.FieldType == "enum" {
			if f.EnumValues == nil {
				return fmt.Errorf("enum_values is required for field_type 'enum': %w", ErrInvalidInput)
			}
			var vals []string
			if err := json.Unmarshal(*f.EnumValues, &vals); err != nil || len(vals) == 0 {
				return fmt.Errorf("enum_values must be a non-empty array: %w", ErrInvalidInput)
			}
		}

		if f.DefaultValue != nil {
			var enumRaw sql.NullString
			if f.FieldType == "enum" && f.EnumValues != nil {
				raw, _ := json.Marshal(f.EnumValues)
				enumRaw = sql.NullString{String: string(raw), Valid: true}
			}
			if err := validateFieldDefaultValue(f.FieldType, f.DefaultValue, enumRaw); err != nil {
				return err
			}
		}

		if seenNames[name] {
			return fmt.Errorf("duplicate field_name '%s': %w", name, ErrInvalidInput)
		}
		seenNames[name] = true
	}
	return nil
}

func validateFieldDefaultValue(fieldType string, value *string, enumRaw sql.NullString) error {
	if value == nil {
		return nil
	}

	switch fieldType {
	case "number":
		var f float64
		if _, err := fmt.Sscanf(*value, "%f", &f); err != nil {
			return fmt.Errorf("default_value for number field must be numeric: %w", ErrInvalidInput)
		}
	case "boolean":
		v := strings.TrimSpace(*value)
		if v != "true" && v != "false" {
			return fmt.Errorf("default_value for boolean field must be 'true' or 'false': %w", ErrInvalidInput)
		}
	case "enum":
		if enumRaw.Valid {
			var enumVals []string
			if err := json.Unmarshal([]byte(enumRaw.String), &enumVals); err == nil {
				found := false
				for _, ev := range enumVals {
					if ev == *value {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("default_value '%s' is not in enum_values: %w", *value, ErrInvalidInput)
				}
			}
		}
	}
	return nil
}

func (s *DefinitionService) resolveLocationForInstance(instanceID string) (string, string) {
	currentID := instanceID
	for i := 0; i < 50; i++ {
		var locationID, locationName sql.NullString
		var parentInstanceID sql.NullString
		err := s.db.QueryRow(`
			SELECT i.location_id, l.name, i.parent_instance_id
			FROM item_instances i
			LEFT JOIN locations l ON l.id = i.location_id
			WHERE i.id = ?
		`, currentID).Scan(&locationID, &locationName, &parentInstanceID)
		if err != nil {
			return "", ""
		}
		if locationID.Valid {
			return locationID.String, locationName.String
		}
		if !parentInstanceID.Valid {
			return "", ""
		}
		currentID = parentInstanceID.String
	}
	return "", ""
}

func (s *DefinitionService) checkFieldNameCollisionWithInherited(defID string, newFieldNames []string) error {
	ancestors, err := s.getAncestorIDs(defID)
	if err != nil {
		return err
	}

	inheritedNames := make(map[string]bool)
	for _, ancID := range ancestors {
		aFields, err := s.getOwnFields(ancID)
		if err != nil {
			return err
		}
		for _, af := range aFields {
			inheritedNames[af.FieldName] = true
		}
	}

	for _, name := range newFieldNames {
		if inheritedNames[name] {
			return fmt.Errorf("field_name '%s' collides with inherited field: %w", name, ErrInvalidInput)
		}
	}

	return nil
}
