package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type CreateInstanceInput struct {
	DefinitionID     string            `json:"definition_id"`
	Quantity         int               `json:"quantity"`
	LocationID       *string           `json:"location_id"`
	ParentInstanceID *string           `json:"parent_instance_id"`
	FieldValues      []FieldValueInput `json:"field_values"`
}

type FieldValueInput struct {
	FieldID string  `json:"field_id"`
	Value   *string `json:"value"`
}

type UpdateInstanceInput struct {
	Quantity    *int              `json:"quantity"`
	FieldValues []FieldValueInput `json:"field_values"`
}

type MoveInstanceInput struct {
	Quantity               int     `json:"quantity"`
	TargetLocationID       *string `json:"target_location_id"`
	TargetParentInstanceID *string `json:"target_parent_instance_id"`
}

type MoveResult struct {
	Source *InstanceDetail `json:"source"`
	Target InstanceDetail  `json:"target"`
}

type InstanceListResult struct {
	Instances  []InstanceSummary `json:"instances"`
	TotalCount int               `json:"total_count"`
	Truncated  bool              `json:"truncated,omitempty"`
}

type InstanceDetail struct {
	ID                 string                `json:"id"`
	DefinitionID       string                `json:"definition_id"`
	DefinitionName     string                `json:"definition_name"`
	ParentDefID        *string               `json:"parent_def_id"`
	ParentDefName      *string               `json:"parent_def_name"`
	Unit               *string               `json:"unit"`
	Quantity           int                   `json:"quantity"`
	LocationID         *string               `json:"location_id"`
	LocationName       *string               `json:"location_name"`
	ParentInstanceID   *string               `json:"parent_instance_id"`
	ParentInstanceName *string               `json:"parent_instance_name"`
	FieldValues        []InstanceFieldValue  `json:"field_values"`
	ChildInstanceCount int                   `json:"child_instance_count"`
	Breadcrumb         []BreadcrumbEntry     `json:"breadcrumb"`
	CreatedAt          string                `json:"created_at"`
	UpdatedAt          string                `json:"updated_at"`
}

type InstanceFieldValue struct {
	FieldID    string           `json:"field_id"`
	FieldName  string           `json:"field_name"`
	FieldType  string           `json:"field_type"`
	EnumValues *json.RawMessage `json:"enum_values"`
	Value      *string          `json:"value"`
}

type BreadcrumbEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type InstanceService struct {
	db *sql.DB
}

func NewInstanceService(db *sql.DB) *InstanceService {
	return &InstanceService{db: db}
}

func (s *InstanceService) Create(ctx interface{}, input CreateInstanceInput) (*InstanceDetail, error) {
	if strings.TrimSpace(input.DefinitionID) == "" {
		return nil, fmt.Errorf("definition_id is required: %w", ErrInvalidInput)
	}
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0: %w", ErrInvalidInput)
	}

	if (input.LocationID == nil) == (input.ParentInstanceID == nil) {
		return nil, fmt.Errorf("exactly one of location_id or parent_instance_id must be provided: %w", ErrInvalidInput)
	}

	var defName string
	var isContainer bool
	err := s.db.QueryRow(`SELECT name, is_container FROM item_definitions WHERE id = ?`, input.DefinitionID).Scan(&defName, &isContainer)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("definition not found: %w", ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get definition: %w", err)
	}

	if input.ParentInstanceID != nil {
		var parentExists bool
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_instances WHERE id = ?)`, *input.ParentInstanceID).Scan(&parentExists)
		if err != nil {
			return nil, fmt.Errorf("check parent instance: %w", err)
		}
		if !parentExists {
			return nil, fmt.Errorf("parent instance not found: %w", ErrNotFound)
		}

		var parentDefIsContainer bool
		err = s.db.QueryRow(`
			SELECT d.is_container FROM item_definitions d
			JOIN item_instances i ON i.definition_id = d.id
			WHERE i.id = ?
		`, *input.ParentInstanceID).Scan(&parentDefIsContainer)
		if err != nil {
			return nil, fmt.Errorf("get parent definition: %w", err)
		}
		if !parentDefIsContainer {
			return nil, fmt.Errorf("parent instance is not a container: %w", ErrInvalidInput)
		}
	}

	if err := s.validateFieldValues(input.DefinitionID, input.FieldValues); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	matchingID := s.findMatchingInstanceTx(tx, input.DefinitionID, input.LocationID, input.ParentInstanceID, input.FieldValues)
	if matchingID != nil {
		_, err := tx.Exec(
			`UPDATE item_instances SET quantity = quantity + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			input.Quantity, *matchingID,
		)
		if err != nil {
			return nil, fmt.Errorf("merge instance: %w", err)
		}
		tx.Commit()
		return s.GetByID(*matchingID)
	}

	id := uuid.New().String()

	_, err = tx.Exec(
		`INSERT INTO item_instances (id, definition_id, quantity, location_id, parent_instance_id) VALUES (?, ?, ?, ?, ?)`,
		id, input.DefinitionID, input.Quantity, input.LocationID, input.ParentInstanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	for _, fv := range input.FieldValues {
		fvID := uuid.New().String()
		_, err := tx.Exec(
			`INSERT INTO instance_field_values (id, instance_id, field_id, value) VALUES (?, ?, ?, ?)`,
			fvID, id, fv.FieldID, fv.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("insert field value: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetByID(id)
}

func (s *InstanceService) GetByID(id string) (*InstanceDetail, error) {
	var inst InstanceDetail
	var locID, parentInstID, unit sql.NullString
	var parentDefID sql.NullString

	err := s.db.QueryRow(`
		SELECT i.id, i.definition_id, d.name, d.parent_def_id, d.unit,
		       i.quantity, i.location_id, i.parent_instance_id,
		       i.created_at, i.updated_at
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		WHERE i.id = ?
	`, id).Scan(&inst.ID, &inst.DefinitionID, &inst.DefinitionName,
		&parentDefID, &unit,
		&inst.Quantity, &locID, &parentInstID,
		&inst.CreatedAt, &inst.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	if unit.Valid {
		inst.Unit = &unit.String
	}

	if parentDefID.Valid {
		inst.ParentDefID = &parentDefID.String
		var pdn string
		err := s.db.QueryRow(`SELECT name FROM item_definitions WHERE id = ?`, parentDefID.String).Scan(&pdn)
		if err == nil {
			inst.ParentDefName = &pdn
		}
	}

	if locID.Valid {
		inst.LocationID = &locID.String
		var locName string
		err := s.db.QueryRow(`SELECT name FROM locations WHERE id = ?`, locID.String).Scan(&locName)
		if err == nil {
			inst.LocationName = &locName
		}
	}

	if parentInstID.Valid {
		inst.ParentInstanceID = &parentInstID.String
		var piDefName string
		err := s.db.QueryRow(`
			SELECT d.name FROM item_definitions d
			JOIN item_instances pi ON pi.definition_id = d.id
			WHERE pi.id = ?
		`, parentInstID.String).Scan(&piDefName)
		if err == nil {
			inst.ParentInstanceName = &piDefName
		}
	}

	fieldValues, err := s.getInstanceFieldValues(id, inst.DefinitionID)
	if err != nil {
		return nil, err
	}
	inst.FieldValues = fieldValues

	s.db.QueryRow(
		`SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = ?`,
		id,
	).Scan(&inst.ChildInstanceCount)

	breadcrumb, err := s.GetBreadcrumb(id)
	if err != nil {
		return nil, err
	}
	inst.Breadcrumb = breadcrumb

	return &inst, nil
}

func (s *InstanceService) getInstanceFieldValues(instanceID, definitionID string) ([]InstanceFieldValue, error) {
	resolvedFields, err := s.resolveFieldSchema(definitionID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT field_id, value FROM instance_field_values WHERE instance_id = ?
	`, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get field values: %w", err)
	}
	defer rows.Close()

	storedValues := make(map[string]*string)
	for rows.Next() {
		var fieldID string
		var value sql.NullString
		if err := rows.Scan(&fieldID, &value); err != nil {
			return nil, fmt.Errorf("scan field value: %w", err)
		}
		if value.Valid {
			val := value.String
			storedValues[fieldID] = &val
		} else {
			storedValues[fieldID] = nil
		}
	}

	var result []InstanceFieldValue
	for _, f := range resolvedFields {
		val, ok := storedValues[f.id]
		entry := InstanceFieldValue{
			FieldID:    f.id,
			FieldName:  f.fieldName,
			FieldType:  f.fieldType,
			EnumValues: f.enumValues,
		}
		if !ok {
			entry.Value = f.defaultValue
		} else {
			entry.Value = val
		}
		result = append(result, entry)
	}

	if result == nil {
		result = []InstanceFieldValue{}
	}

	return result, nil
}

type resolvedField struct {
	id           string
	fieldName    string
	fieldType    string
	enumValues   *json.RawMessage
	isRequired   bool
	defaultValue *string
}

func (s *InstanceService) resolveFieldSchema(definitionID string) ([]resolvedField, error) {
	var fields []resolvedField

	ancestors, err := s.getDefinitionAncestors(definitionID)
	if err != nil {
		return nil, err
	}

	ownFields, err := s.getDefinitionOwnFields(definitionID)
	if err != nil {
		return nil, err
	}
	fields = append(fields, ownFields...)

	for _, ancID := range ancestors {
		af, err := s.getDefinitionOwnFields(ancID)
		if err != nil {
			return nil, err
		}
		fields = append(fields, af...)
	}

	return fields, nil
}

func (s *InstanceService) getDefinitionAncestors(defID string) ([]string, error) {
	var ancestors []string
	currentID := defID

	for i := 0; i < 50; i++ {
		var parentID sql.NullString
		err := s.db.QueryRow(`SELECT parent_def_id FROM item_definitions WHERE id = ?`, currentID).Scan(&parentID)
		if err != nil {
			return nil, fmt.Errorf("get parent definition: %w", err)
		}
		if !parentID.Valid {
			break
		}
		ancestors = append(ancestors, parentID.String)
		currentID = parentID.String
	}

	return ancestors, nil
}

func (s *InstanceService) getDefinitionOwnFields(defID string) ([]resolvedField, error) {
	rows, err := s.db.Query(`
		SELECT id, field_name, field_type, enum_values, is_required, default_value
		FROM definition_fields
		WHERE definition_id = ?
		ORDER BY display_order ASC
	`, defID)
	if err != nil {
		return nil, fmt.Errorf("get definition fields: %w", err)
	}
	defer rows.Close()

	var fields []resolvedField
	for rows.Next() {
		var f resolvedField
		var enumRaw sql.NullString
		if err := rows.Scan(&f.id, &f.fieldName, &f.fieldType, &enumRaw, &f.isRequired, &f.defaultValue); err != nil {
			return nil, fmt.Errorf("scan field: %w", err)
		}
		if enumRaw.Valid {
			raw := json.RawMessage(enumRaw.String)
			f.enumValues = &raw
		}
		fields = append(fields, f)
	}

	return fields, rows.Err()
}

func (s *InstanceService) validateFieldValues(definitionID string, fieldValues []FieldValueInput) error {
	resolvedFields, err := s.resolveFieldSchema(definitionID)
	if err != nil {
		return err
	}

	fieldMap := make(map[string]resolvedField)
	for _, f := range resolvedFields {
		fieldMap[f.id] = f
	}

	for _, fv := range fieldValues {
		field, ok := fieldMap[fv.FieldID]
		if !ok {
			return fmt.Errorf("field_id '%s' does not belong to this definition's schema: %w", fv.FieldID, ErrInvalidInput)
		}
		if fv.Value != nil {
			if err := s.validateFieldValue(field, *fv.Value); err != nil {
				return err
			}
		}
	}

	providedFieldIDs := make(map[string]bool)
	for _, fv := range fieldValues {
		providedFieldIDs[fv.FieldID] = true
	}

	for _, f := range resolvedFields {
		if f.isRequired && !providedFieldIDs[f.id] && f.defaultValue == nil {
			return fmt.Errorf("required field '%s' is missing: %w", f.fieldName, ErrInvalidInput)
		}
	}

	return nil
}

func (s *InstanceService) validateFieldValue(field resolvedField, value string) error {
	switch field.fieldType {
	case "number":
		if _, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err != nil {
			return fmt.Errorf("value for field '%s' must be a number: %w", field.fieldName, ErrInvalidInput)
		}
	case "boolean":
		v := strings.TrimSpace(value)
		if v != "true" && v != "false" {
			return fmt.Errorf("value for field '%s' must be 'true' or 'false': %w", field.fieldName, ErrInvalidInput)
		}
	case "enum":
		if field.enumValues == nil {
			return fmt.Errorf("field '%s' has no enum_values defined: %w", field.fieldName, ErrInvalidInput)
		}
		var rawArray []string
		if err := json.Unmarshal(*field.enumValues, &rawArray); err != nil {
			return fmt.Errorf("invalid enum_values for field '%s': %w", field.fieldName, ErrInvalidInput)
		}
		v := strings.TrimSpace(value)
		found := false
		for _, ev := range rawArray {
			if ev == v {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value '%s' is not in enum_values for field '%s': %w", v, field.fieldName, ErrInvalidInput)
		}
	}

	return nil
}

func (s *InstanceService) findMatchingInstanceTx(tx *sql.Tx, definitionID string, locationID, parentInstanceID *string, fieldValues []FieldValueInput) *string {
	var query string
	var args []interface{}

	if locationID != nil {
		query = `SELECT id FROM item_instances WHERE definition_id = ? AND location_id = ? AND parent_instance_id IS NULL`
		args = append(args, definitionID, *locationID)
	} else {
		query = `SELECT id FROM item_instances WHERE definition_id = ? AND parent_instance_id = ? AND location_id IS NULL`
		args = append(args, definitionID, *parentInstanceID)
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var candidateIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		candidateIDs = append(candidateIDs, id)
	}
	rows.Close()

	for _, candidateID := range candidateIDs {
		var storedCount int
		tx.QueryRow(`SELECT COUNT(*) FROM instance_field_values WHERE instance_id = ?`, candidateID).Scan(&storedCount)

		if len(fieldValues) == 0 && storedCount == 0 {
			return &candidateID
		}

		if storedCount != len(fieldValues) {
			continue
		}

		allMatch := true
		for _, fv := range fieldValues {
			var storedValue sql.NullString
			err := tx.QueryRow(`SELECT value FROM instance_field_values WHERE instance_id = ? AND field_id = ?`, candidateID, fv.FieldID).Scan(&storedValue)
			if err != nil {
				allMatch = false
				break
			}

			if fv.Value == nil && !storedValue.Valid {
				continue
			}
			if fv.Value == nil || !storedValue.Valid {
				allMatch = false
				break
			}
			if *fv.Value != storedValue.String {
				allMatch = false
				break
			}
		}

		if allMatch {
			return &candidateID
		}
	}

	return nil
}

func (s *InstanceService) Update(id string, input UpdateInstanceInput) (*InstanceDetail, error) {
	existing, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if input.Quantity != nil {
		if *input.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be greater than 0: %w", ErrInvalidInput)
		}
		_, err = tx.Exec(
			`UPDATE item_instances SET quantity = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			*input.Quantity, id,
		)
		if err != nil {
			return nil, fmt.Errorf("update quantity: %w", err)
		}
	}

	if input.FieldValues != nil {
		if err := s.validateFieldValues(existing.DefinitionID, input.FieldValues); err != nil {
			return nil, err
		}

		if _, err := tx.Exec(`DELETE FROM instance_field_values WHERE instance_id = ?`, id); err != nil {
			return nil, fmt.Errorf("clear field values: %w", err)
		}

		for _, fv := range input.FieldValues {
			fvID := uuid.New().String()
			_, err := tx.Exec(
				`INSERT INTO instance_field_values (id, instance_id, field_id, value) VALUES (?, ?, ?, ?)`,
				fvID, id, fv.FieldID, fv.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("insert field value: %w", err)
			}
		}
	}

	if input.Quantity == nil && input.FieldValues == nil {
		return existing, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.GetByID(id)
}

func (s *InstanceService) Delete(id string) error {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = ?`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("count children: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("Cannot delete: %d items are stored inside this instance. Move them out first.", count)
	}

	result, err := s.db.Exec(`DELETE FROM item_instances WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete instance: %w", err)
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

func (s *InstanceService) Move(sourceID string, input MoveInstanceInput) (*MoveResult, error) {
	source, err := s.getBaseInstance(sourceID)
	if err != nil {
		return nil, err
	}

	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0: %w", ErrInvalidInput)
	}
	if input.Quantity > source.Quantity {
		return nil, fmt.Errorf("Cannot move %d items: only %d available.", input.Quantity, source.Quantity)
	}

	if (input.TargetLocationID == nil) == (input.TargetParentInstanceID == nil) {
		return nil, fmt.Errorf("exactly one of target_location_id or target_parent_instance_id must be provided: %w", ErrInvalidInput)
	}

	if input.TargetParentInstanceID != nil {
		if *input.TargetParentInstanceID == sourceID {
			return nil, fmt.Errorf("cannot move instance into itself: %w", ErrInvalidInput)
		}

		var targetExists bool
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_instances WHERE id = ?)`, *input.TargetParentInstanceID).Scan(&targetExists)
		if err != nil {
			return nil, fmt.Errorf("check target parent: %w", err)
		}
		if !targetExists {
			return nil, fmt.Errorf("target parent instance not found: %w", ErrNotFound)
		}

		var tgtDefIsContainer bool
		err = s.db.QueryRow(`
			SELECT d.is_container FROM item_definitions d
			JOIN item_instances i ON i.definition_id = d.id
			WHERE i.id = ?
		`, *input.TargetParentInstanceID).Scan(&tgtDefIsContainer)
		if err != nil {
			return nil, fmt.Errorf("get target parent definition: %w", err)
		}
		if !tgtDefIsContainer {
			return nil, fmt.Errorf("target parent instance is not a container: %w", ErrInvalidInput)
		}

		if s.isAncestorInstance(sourceID, *input.TargetParentInstanceID) {
			return nil, fmt.Errorf("cycle detected: target is a child of the source: %w", ErrInvalidInput)
		}
	}

	if input.TargetLocationID != nil {
		var tgtLocExists bool
		err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM locations WHERE id = ?)`, *input.TargetLocationID).Scan(&tgtLocExists)
		if err != nil {
			return nil, fmt.Errorf("check target location: %w", err)
		}
		if !tgtLocExists {
			return nil, fmt.Errorf("target location not found: %w", ErrNotFound)
		}
	}

	if source.LocationID != nil && input.TargetLocationID != nil && *source.LocationID == *input.TargetLocationID {
		return nil, fmt.Errorf("instance is already at this location: %w", ErrInvalidInput)
	}
	if source.ParentInstanceID != nil && input.TargetParentInstanceID != nil && *source.ParentInstanceID == *input.TargetParentInstanceID {
		return nil, fmt.Errorf("instance is already in this container: %w", ErrInvalidInput)
	}

	sourceFieldValues, err := s.getStoredFieldValues(sourceID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	newQuantity := source.Quantity - input.Quantity
	var sourceResult *InstanceDetail

	if newQuantity == 0 {
		var childCount int
		tx.QueryRow(`SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = ?`, sourceID).Scan(&childCount)
		if childCount > 0 {
			return nil, fmt.Errorf("Cannot move all items: source instance has %d children. Move them out first.", childCount)
		}

		_, err = tx.Exec(`DELETE FROM item_instances WHERE id = ?`, sourceID)
		if err != nil {
			return nil, fmt.Errorf("delete exhausted source: %w", err)
		}
		sourceResult = nil
	} else {
		_, err = tx.Exec(
			`UPDATE item_instances SET quantity = quantity - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			input.Quantity, sourceID,
		)
		if err != nil {
			return nil, fmt.Errorf("decrement source: %w", err)
		}
	}

	var targetID string
	matchingID := s.findMatchingInstanceTx(tx, source.DefinitionID, input.TargetLocationID, input.TargetParentInstanceID, sourceFieldValues)
	if matchingID != nil {
		_, err := tx.Exec(
			`UPDATE item_instances SET quantity = quantity + ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			input.Quantity, *matchingID,
		)
		if err != nil {
			return nil, fmt.Errorf("merge at target: %w", err)
		}
		targetID = *matchingID
	} else {
		targetID = uuid.New().String()
		_, err = tx.Exec(
			`INSERT INTO item_instances (id, definition_id, quantity, location_id, parent_instance_id) VALUES (?, ?, ?, ?, ?)`,
			targetID, source.DefinitionID, input.Quantity, input.TargetLocationID, input.TargetParentInstanceID,
		)
		if err != nil {
			return nil, fmt.Errorf("create target instance: %w", err)
		}

		for _, fv := range sourceFieldValues {
			fvID := uuid.New().String()
			_, err = tx.Exec(
				`INSERT INTO instance_field_values (id, instance_id, field_id, value) VALUES (?, ?, ?, ?)`,
				fvID, targetID, fv.FieldID, fv.Value,
			)
			if err != nil {
				return nil, fmt.Errorf("insert target field value: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	targetDetail, err := s.GetByID(targetID)
	if err != nil {
		return nil, err
	}

	if newQuantity > 0 {
		sourceDetail, err := s.GetByID(sourceID)
		if err != nil {
			return nil, err
		}
		sourceResult = sourceDetail
	}

	return &MoveResult{
		Source: sourceResult,
		Target: *targetDetail,
	}, nil
}

type baseInstance struct {
	ID             string
	DefinitionID   string
	Quantity       int
	LocationID     *string
	ParentInstanceID *string
}

func (s *InstanceService) getBaseInstance(id string) (*baseInstance, error) {
	var inst baseInstance
	var locID, parentInstID sql.NullString
	err := s.db.QueryRow(`
		SELECT id, definition_id, quantity, location_id, parent_instance_id
		FROM item_instances WHERE id = ?
	`, id).Scan(&inst.ID, &inst.DefinitionID, &inst.Quantity, &locID, &parentInstID)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get base instance: %w", err)
	}

	if locID.Valid {
		inst.LocationID = &locID.String
	}
	if parentInstID.Valid {
		inst.ParentInstanceID = &parentInstID.String
	}

	return &inst, nil
}

func (s *InstanceService) getStoredFieldValues(instanceID string) ([]FieldValueInput, error) {
	rows, err := s.db.Query(
		`SELECT field_id, value FROM instance_field_values WHERE instance_id = ?`,
		instanceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get stored field values: %w", err)
	}
	defer rows.Close()

	var values []FieldValueInput
	for rows.Next() {
		var fv FieldValueInput
		var val sql.NullString
		if err := rows.Scan(&fv.FieldID, &val); err != nil {
			return nil, fmt.Errorf("scan stored field: %w", err)
		}
		if val.Valid {
			fv.Value = &val.String
		}
		values = append(values, fv)
	}

	return values, rows.Err()
}

func (s *InstanceService) isAncestorInstance(instanceID, potentialDescendant string) bool {
	currentID := potentialDescendant
	for i := 0; i < 50; i++ {
		if currentID == instanceID {
			return true
		}
		var parentID sql.NullString
		err := s.db.QueryRow(`SELECT parent_instance_id FROM item_instances WHERE id = ?`, currentID).Scan(&parentID)
		if err != nil || !parentID.Valid {
			return false
		}
		currentID = parentID.String
	}
	return false
}

func (s *InstanceService) List(locationID, definitionID, parentInstanceID *string) (*InstanceListResult, error) {
	var whereParts []string
	var args []interface{}

	if locationID != nil {
		whereParts = append(whereParts, "i.location_id = ?")
		args = append(args, *locationID)
	}
	if definitionID != nil {
		whereParts = append(whereParts, "i.definition_id = ?")
		args = append(args, *definitionID)
	}
	if parentInstanceID != nil {
		whereParts = append(whereParts, "i.parent_instance_id = ?")
		args = append(args, *parentInstanceID)
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereParts, " AND ")
	}

	var totalCount int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM item_instances i"+whereClause,
		args...,
	).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("count instances: %w", err)
	}

	query := `
		SELECT i.id, i.definition_id, d.name, i.quantity,
		       i.location_id, l.name, i.parent_instance_id, pd.name, i.updated_at
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		LEFT JOIN locations l ON l.id = i.location_id
		LEFT JOIN item_instances pi ON pi.id = i.parent_instance_id
		LEFT JOIN item_definitions pd ON pd.id = pi.definition_id` + whereClause + `
		ORDER BY i.updated_at DESC
		LIMIT 500
	`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer rows.Close()

	var instances []InstanceSummary
	for rows.Next() {
		var is InstanceSummary
		var locID, locName, parentID, parentName sql.NullString
		if err := rows.Scan(&is.ID, &is.DefinitionID, &is.DefinitionName, &is.Quantity,
			&locID, &locName, &parentID, &parentName, &is.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan instance summary: %w", err)
		}

		if locID.Valid {
			is.LocationID = &locID.String
		}
		if locName.Valid {
			is.LocationName = &locName.String
		}
		if parentID.Valid {
			is.ParentInstanceID = &parentID.String
		}
		if parentName.Valid {
			is.ParentInstanceName = &parentName.String
		}

		instances = append(instances, is)
	}

	if instances == nil {
		instances = []InstanceSummary{}
	}

	result := &InstanceListResult{
		Instances:  instances,
		TotalCount: totalCount,
	}

	if totalCount > 500 {
		result.Truncated = true
	}

	return result, rows.Err()
}

func (s *InstanceService) GetContents(id string) ([]InstanceSummary, error) {
	var exists bool
	s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_instances WHERE id = ?)`, id).Scan(&exists)
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.db.Query(`
		SELECT i.id, i.definition_id, d.name, i.quantity
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		WHERE i.parent_instance_id = ?
		ORDER BY d.name ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get contents: %w", err)
	}
	defer rows.Close()

	var instances []InstanceSummary
	for rows.Next() {
		var is InstanceSummary
		if err := rows.Scan(&is.ID, &is.DefinitionID, &is.DefinitionName, &is.Quantity); err != nil {
			return nil, fmt.Errorf("scan content instance: %w", err)
		}
		is.ParentInstanceID = &id
		instances = append(instances, is)
	}

	if instances == nil {
		instances = []InstanceSummary{}
	}

	return instances, rows.Err()
}

func (s *InstanceService) GetBreadcrumb(id string) ([]BreadcrumbEntry, error) {
	var exists bool
	s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_instances WHERE id = ?)`, id).Scan(&exists)
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.db.Query(`
		WITH RECURSIVE instance_chain AS (
			SELECT i.id, d.name AS definition_name, i.quantity, i.parent_instance_id, i.location_id, 0 AS depth
			FROM item_instances i
			JOIN item_definitions d ON d.id = i.definition_id
			WHERE i.id = ?
			UNION ALL
			SELECT i.id, d.name, i.quantity, i.parent_instance_id, i.location_id, c.depth + 1
			FROM item_instances i
			JOIN item_definitions d ON d.id = i.definition_id
			JOIN instance_chain c ON i.id = c.parent_instance_id
		)
		SELECT id, definition_name, quantity, location_id, depth FROM instance_chain ORDER BY depth ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get instance chain: %w", err)
	}
	defer rows.Close()

	type chainEntry struct {
		id             string
		definitionName string
		quantity       int
		locationID     sql.NullString
		depth          int
	}

	var chain []chainEntry
	for rows.Next() {
		var ce chainEntry
		var qty int64
		if err := rows.Scan(&ce.id, &ce.definitionName, &qty, &ce.locationID, &ce.depth); err != nil {
			return nil, fmt.Errorf("scan chain entry: %w", err)
		}
		ce.quantity = int(qty)
		chain = append(chain, ce)
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("instance chain is empty: %w", ErrNotFound)
	}

	var locationID string
	var instanceEntries []chainEntry
	for _, ce := range chain {
		if ce.locationID.Valid && locationID == "" {
			locationID = ce.locationID.String
		}
		if ce.depth > 0 || !ce.locationID.Valid {
			instanceEntries = append(instanceEntries, ce)
		}
	}

	var breadcrumb []BreadcrumbEntry

	if locationID != "" {
		locRows, err := s.db.Query(`
			WITH RECURSIVE ancestors AS (
				SELECT id, name, parent_id, 0 AS depth FROM locations WHERE id = ?
				UNION ALL
				SELECT l.id, l.name, l.parent_id, a.depth + 1
				FROM locations l JOIN ancestors a ON l.id = a.parent_id
			)
			SELECT id, name FROM ancestors ORDER BY depth DESC
		`, locationID)
		if err != nil {
			return nil, fmt.Errorf("get location breadcrumb: %w", err)
		}
		defer locRows.Close()

		for locRows.Next() {
			var be BreadcrumbEntry
			if err := locRows.Scan(&be.ID, &be.Name); err != nil {
				return nil, fmt.Errorf("scan location entry: %w", err)
			}
			be.Kind = "location"
			breadcrumb = append(breadcrumb, be)
		}
	}

	for _, ce := range instanceEntries {
		breadcrumb = append(breadcrumb, BreadcrumbEntry{
			ID:   ce.id,
			Name: ce.definitionName + " (x" + strconv.Itoa(ce.quantity) + ")",
			Kind: "instance",
		})
	}

	requestedInstance := chain[0]
	breadcrumb = append(breadcrumb, BreadcrumbEntry{
		ID:   requestedInstance.id,
		Name: requestedInstance.definitionName + " (x" + strconv.Itoa(requestedInstance.quantity) + ")",
		Kind: "instance",
	})

	if breadcrumb == nil {
		breadcrumb = []BreadcrumbEntry{}
	}

	return breadcrumb, nil
}

func (s *InstanceService) ResolveLocationForInstance(instanceID string) (string, string) {
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
