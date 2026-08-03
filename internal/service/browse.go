package service

import (
	"database/sql"
	"fmt"
	"sort"
)

type BrowseNode struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    *string       `json:"description"`
	Kind           string        `json:"kind"`
	Children       []BrowseNode  `json:"children"`
	Stacks         []BrowseStack `json:"stacks"`
	StackCount     int           `json:"stack_count"`
	StackTruncated bool          `json:"stack_truncated"`
}

type rawStack struct {
	DefinitionID     string
	DefinitionName   string
	Unit             sql.NullString
	IsContainer      bool
	TotalQuantity    int
	InstanceCount    int
	LocationID       string
	FirstInstanceID  sql.NullString
}

type nodeAux struct {
	id          string
	name        string
	description *string
	parentID    *string
	stacks      []BrowseStack
	stackCount  int
	stackTrunc  bool
	children    []string
}

type BrowseService struct {
	db *sql.DB
}

func NewBrowseService(db *sql.DB) *BrowseService {
	return &BrowseService{db: db}
}

const browseStackLimit = 50

func (s *BrowseService) GetBrowse() ([]BrowseNode, error) {
	nodeMap := make(map[string]*nodeAux)
	var rootIDs []string

	locRows, err := s.db.Query(
		`SELECT id, name, description, parent_id FROM locations ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("get browse locations: %w", err)
	}
	defer locRows.Close()

	for locRows.Next() {
		var id, name string
		var description, parentID *string
		if err := locRows.Scan(&id, &name, &description, &parentID); err != nil {
			return nil, fmt.Errorf("scan browse location: %w", err)
		}

		nodeMap[id] = &nodeAux{
			id:          id,
			name:        name,
			description: description,
			parentID:    parentID,
		}

		if parentID == nil {
			rootIDs = append(rootIDs, id)
		}
	}
	if err := locRows.Err(); err != nil {
		return nil, err
	}

	for _, node := range nodeMap {
		if node.parentID != nil {
			if parent, ok := nodeMap[*node.parentID]; ok {
				parent.children = append(parent.children, node.id)
			}
		}
	}

	for _, node := range nodeMap {
		sort.Slice(node.children, func(i, j int) bool {
			return nodeMap[node.children[i]].name < nodeMap[node.children[j]].name
		})
	}

	stackRows, err := s.db.Query(`
		SELECT
			i.location_id,
			d.id AS definition_id,
			d.name AS definition_name,
			d.unit,
			d.is_container,
			COALESCE(SUM(i.quantity), 0) AS total_quantity,
			COUNT(i.id) AS instance_count,
			MIN(i.id) AS first_instance_id
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		WHERE i.location_id IS NOT NULL
		GROUP BY i.location_id, d.id, i.parent_instance_id
		ORDER BY i.location_id, d.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("get browse stacks: %w", err)
	}
	defer stackRows.Close()

	var allStacks []rawStack
	for stackRows.Next() {
		var rs rawStack
		if err := stackRows.Scan(&rs.LocationID, &rs.DefinitionID, &rs.DefinitionName, &rs.Unit, &rs.IsContainer, &rs.TotalQuantity, &rs.InstanceCount, &rs.FirstInstanceID); err != nil {
			return nil, fmt.Errorf("scan browse stack: %w", err)
		}
		allStacks = append(allStacks, rs)
	}
	if err := stackRows.Err(); err != nil {
		return nil, err
	}

	perLocation := make(map[string][]rawStack)
	for _, rs := range allStacks {
		perLocation[rs.LocationID] = append(perLocation[rs.LocationID], rs)
	}

	for locID, rsList := range perLocation {
		node, ok := nodeMap[locID]
		if !ok {
			continue
		}
		total := len(rsList)
		truncated := total > browseStackLimit
		take := total
		if take > browseStackLimit {
			take = browseStackLimit
		}
		var stacks []BrowseStack
		for _, rs := range rsList[:take] {
			stack := BrowseStack{
				DefinitionID:   rs.DefinitionID,
				DefinitionName: rs.DefinitionName,
				TotalQuantity:  rs.TotalQuantity,
				InstanceCount:  rs.InstanceCount,
				IsContainer:    rs.IsContainer,
			}
			if rs.Unit.Valid {
				stack.Unit = &rs.Unit.String
			}
			if rs.FirstInstanceID.Valid && rs.InstanceCount == 1 {
				stack.SingleInstanceID = &rs.FirstInstanceID.String
			}
			stacks = append(stacks, stack)
		}
		node.stacks = stacks
		node.stackCount = total
		node.stackTrunc = truncated
	}

	var buildNode func(nid string) BrowseNode
	buildNode = func(nid string) BrowseNode {
		aux := nodeMap[nid]
		children := make([]BrowseNode, 0, len(aux.children))
		for _, cid := range aux.children {
			children = append(children, buildNode(cid))
		}
		stacks := aux.stacks
		if stacks == nil {
			stacks = []BrowseStack{}
		}
		return BrowseNode{
			ID:             aux.id,
			Name:           aux.name,
			Description:    aux.description,
			Kind:           "location",
			Children:       children,
			Stacks:         stacks,
			StackCount:     aux.stackCount,
			StackTruncated: aux.stackTrunc,
		}
	}

	roots := make([]BrowseNode, 0, len(rootIDs))
	for _, rid := range rootIDs {
		roots = append(roots, buildNode(rid))
	}

	return roots, nil
}
