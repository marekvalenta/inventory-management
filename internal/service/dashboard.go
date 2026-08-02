package service

import (
	"context"
	"database/sql"
	"sort"
)

type DashboardService struct {
	db *sql.DB
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{db: db}
}

type DashboardData struct {
	Stats        DashboardStats `json:"stats"`
	Locations    []LocationNode `json:"locations"`
	IsOnboarding bool           `json:"is_onboarding"`
}

type DashboardStats struct {
	LocationsCount   int `json:"locations_count"`
	DefinitionsCount int `json:"definitions_count"`
	InstancesCount   int `json:"instances_count"`
	TotalQuantity    int `json:"total_quantity"`
}

type LocationNode struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	InstanceCount      int            `json:"instance_count"`
	DirectInstanceCount int           `json:"direct_instance_count"`
	SubLocationCount   int            `json:"sub_location_count"`
	Children           []LocationNode `json:"children"`
}

func (s *DashboardService) GetDashboard(ctx context.Context) (*DashboardData, error) {
	var rootLocationID string
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(root_location_id, '') FROM settings WHERE id = 1").Scan(&rootLocationID)
	if err != nil {
		return s.emptyDashboard(), nil
	}

	stats, err := s.fetchStats(ctx)
	if err != nil {
		return nil, err
	}

	locRows, err := s.db.QueryContext(ctx, "SELECT id, name, parent_id FROM locations")
	if err != nil {
		return nil, err
	}
	defer locRows.Close()

	type locRow struct {
		ID       string
		Name     string
		ParentID *string
	}
	locations := make(map[string]*locRow)
	for locRows.Next() {
		lr := &locRow{}
		if err := locRows.Scan(&lr.ID, &lr.Name, &lr.ParentID); err != nil {
			return nil, err
		}
		locations[lr.ID] = lr
	}

	countRows, err := s.db.QueryContext(ctx,
		"SELECT location_id, SUM(quantity), COUNT(*) FROM item_instances WHERE location_id IS NOT NULL GROUP BY location_id")
	if err != nil {
		return nil, err
	}
	defer countRows.Close()

	directQty := make(map[string]int)
	for countRows.Next() {
		var locID string
		var qty int
		var dummy int
		if err := countRows.Scan(&locID, &qty, &dummy); err != nil {
			return nil, err
		}
		directQty[locID] = qty
	}

	nodes := make(map[string]*LocationNode)
	for id, loc := range locations {
		nodes[id] = &LocationNode{
			ID:                 id,
			Name:               loc.Name,
			DirectInstanceCount: directQty[id],
			InstanceCount:      directQty[id],
			Children:           []LocationNode{},
		}
	}

	childrenMap := make(map[string][]string)
	for id, loc := range locations {
		if loc.ParentID != nil {
			parentID := *loc.ParentID
			childrenMap[parentID] = append(childrenMap[parentID], id)
		}
	}

	for id := range locations {
		s.computeRecursive(nodes, childrenMap, id)
	}

	for parentID, childIDs := range childrenMap {
		parent := nodes[parentID]
		parent.SubLocationCount = len(childIDs)
		for _, childID := range childIDs {
			child := nodes[childID]
			parent.Children = append(parent.Children, *child)
		}
		sort.Slice(parent.Children, func(i, j int) bool {
			return parent.Children[i].InstanceCount > parent.Children[j].InstanceCount
		})
		if len(parent.Children) > 3 {
			parent.Children = parent.Children[:3]
		}
	}

	var ranked []LocationNode
	for id, node := range nodes {
		loc := locations[id]
		if id == rootLocationID {
			continue
		}
		if loc.ParentID == nil {
			ranked = append(ranked, *node)
		}
	}
	if ranked == nil {
		ranked = []LocationNode{}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].InstanceCount > ranked[j].InstanceCount
	})

	isOnboarding := stats.LocationsCount <= 1 && stats.DefinitionsCount == 0 && stats.InstancesCount == 0

	return &DashboardData{
		Stats:        *stats,
		Locations:    ranked,
		IsOnboarding: isOnboarding,
	}, nil
}

func (s *DashboardService) fetchStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM locations").Scan(&stats.LocationsCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM item_definitions").Scan(&stats.DefinitionsCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM item_instances").Scan(&stats.InstancesCount)
	if err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity), 0) FROM item_instances").Scan(&stats.TotalQuantity)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (s *DashboardService) computeRecursive(nodes map[string]*LocationNode, childrenMap map[string][]string, id string) int {
	node := nodes[id]
	total := node.InstanceCount
	for _, childID := range childrenMap[id] {
		total += s.computeRecursive(nodes, childrenMap, childID)
	}
	node.InstanceCount = total
	return total
}

func (s *DashboardService) emptyDashboard() *DashboardData {
	return &DashboardData{
		Stats:        DashboardStats{},
		Locations:    []LocationNode{},
		IsOnboarding: true,
	}
}
