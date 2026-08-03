package service

import (
	"database/sql"
	"fmt"
	"sort"
)

type BrowseInstance struct {
	ID             string `json:"id"`
	DefinitionID   string `json:"definition_id"`
	DefinitionName string `json:"definition_name"`
	Quantity       int    `json:"quantity"`
	IsContainer    bool   `json:"is_container"`
	ChildCount     int    `json:"child_count"`
}

type BrowseNode struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Description       *string          `json:"description"`
	Kind              string           `json:"kind"`
	Children          []BrowseNode     `json:"children"`
	Instances         []BrowseInstance `json:"instances"`
	InstanceCount     int              `json:"instance_count"`
	InstanceTruncated bool             `json:"instance_truncated"`
}

type rawInstance struct {
	ID             string
	DefinitionID   string
	DefinitionName string
	Quantity       int
	IsContainer    bool
	ChildCount     int
	LocationID     string
}

type nodeAux struct {
	id          string
	name        string
	description *string
	parentID    *string
	instances   []BrowseInstance
	instCount   int
	instTrunc   bool
	children    []string
}

type BrowseService struct {
	db *sql.DB
}

func NewBrowseService(db *sql.DB) *BrowseService {
	return &BrowseService{db: db}
}

const browseInstanceLimit = 50

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

	instRows, err := s.db.Query(`
		SELECT i.id, i.definition_id, d.name, i.quantity, d.is_container,
		       i.location_id,
		       (SELECT COUNT(*) FROM item_instances WHERE parent_instance_id = i.id) as child_count
		FROM item_instances i
		JOIN item_definitions d ON d.id = i.definition_id
		WHERE i.location_id IS NOT NULL
		ORDER BY i.location_id, d.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("get browse instances: %w", err)
	}
	defer instRows.Close()

	var allInstances []rawInstance
	for instRows.Next() {
		var ri rawInstance
		if err := instRows.Scan(&ri.ID, &ri.DefinitionID, &ri.DefinitionName, &ri.Quantity, &ri.IsContainer, &ri.LocationID, &ri.ChildCount); err != nil {
			return nil, fmt.Errorf("scan browse instance: %w", err)
		}
		allInstances = append(allInstances, ri)
	}
	if err := instRows.Err(); err != nil {
		return nil, err
	}

	perLocation := make(map[string][]rawInstance)
	for _, ri := range allInstances {
		perLocation[ri.LocationID] = append(perLocation[ri.LocationID], ri)
	}

	for locID, riList := range perLocation {
		node, ok := nodeMap[locID]
		if !ok {
			continue
		}
		total := len(riList)
		truncated := total > browseInstanceLimit
		take := total
		if take > browseInstanceLimit {
			take = browseInstanceLimit
		}
		var insts []BrowseInstance
		for _, ri := range riList[:take] {
			insts = append(insts, BrowseInstance{
				ID:             ri.ID,
				DefinitionID:   ri.DefinitionID,
				DefinitionName: ri.DefinitionName,
				Quantity:       ri.Quantity,
				IsContainer:    ri.IsContainer,
				ChildCount:     ri.ChildCount,
			})
		}
		node.instances = insts
		node.instCount = total
		node.instTrunc = truncated
	}

	var buildNode func(nid string) BrowseNode
	buildNode = func(nid string) BrowseNode {
		aux := nodeMap[nid]
		children := make([]BrowseNode, 0, len(aux.children))
		for _, cid := range aux.children {
			children = append(children, buildNode(cid))
		}
		insts := aux.instances
		if insts == nil {
			insts = []BrowseInstance{}
		}
		return BrowseNode{
			ID:                aux.id,
			Name:              aux.name,
			Description:       aux.description,
			Kind:              "location",
			Children:          children,
			Instances:         insts,
			InstanceCount:     aux.instCount,
			InstanceTruncated: aux.instTrunc,
		}
	}

	roots := make([]BrowseNode, 0, len(rootIDs))
	for _, rid := range rootIDs {
		roots = append(roots, buildNode(rid))
	}

	return roots, nil
}
