//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/marekvalenta/inventory-management/internal/testutil"
	"github.com/stretchr/testify/require"
)

type dashboardStatsResponse struct {
	LocationsCount   int `json:"locations_count"`
	DefinitionsCount int `json:"definitions_count"`
	InstancesCount   int `json:"instances_count"`
	TotalQuantity    int `json:"total_quantity"`
}

type dashboardLocationNodeResponse struct {
	ID                 string                            `json:"id"`
	Name               string                            `json:"name"`
	InstanceCount      int                               `json:"instance_count"`
	DirectInstanceCount int                              `json:"direct_instance_count"`
	SubLocationCount   int                               `json:"sub_location_count"`
	Children           []dashboardLocationNodeResponse   `json:"children"`
}

type dashboardDataResponse struct {
	Stats        dashboardStatsResponse          `json:"stats"`
	Locations    []dashboardLocationNodeResponse `json:"locations"`
	IsOnboarding bool                            `json:"is_onboarding"`
}

func TestDashboardEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	require.Equal(t, 0, data.Stats.LocationsCount)
	require.Equal(t, 0, data.Stats.DefinitionsCount)
	require.Equal(t, 0, data.Stats.InstancesCount)
	require.Equal(t, 0, data.Stats.TotalQuantity)
	require.Empty(t, data.Locations)
	require.True(t, data.IsOnboarding)
}

func TestDashboardOnboardingState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	rootID, _ := testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	require.Equal(t, 1, data.Stats.LocationsCount)
	require.Equal(t, 0, data.Stats.DefinitionsCount)
	require.Equal(t, 0, data.Stats.InstancesCount)
	require.Equal(t, 0, data.Stats.TotalQuantity)
	require.True(t, data.IsOnboarding)

	_ = rootID
}

func TestDashboardWithData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	_, _ = testutil.SeedRootLocation(t, db)

	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc1', 'Workshop', NULL)`)
	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc2', 'Cabinet', 'loc1')`)
	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc3', 'Garage', NULL)`)
	db.Exec(`INSERT INTO item_definitions (id, name) VALUES ('def1', 'Screw')`)
	db.Exec(`INSERT INTO item_instances (id, definition_id, quantity, location_id) VALUES ('inst1', 'def1', 10, 'loc1')`)
	db.Exec(`INSERT INTO item_instances (id, definition_id, quantity, location_id) VALUES ('inst2', 'def1', 5, 'loc2')`)
	db.Exec(`INSERT INTO item_instances (id, definition_id, quantity, location_id) VALUES ('inst3', 'def1', 3, 'loc3')`)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	require.Equal(t, 4, data.Stats.LocationsCount)
	require.Equal(t, 1, data.Stats.DefinitionsCount)
	require.Equal(t, 3, data.Stats.InstancesCount)
	require.Equal(t, 18, data.Stats.TotalQuantity)
	require.False(t, data.IsOnboarding)

	require.Len(t, data.Locations, 2)

	var workshop, garage *dashboardLocationNodeResponse
	for i := range data.Locations {
		switch data.Locations[i].Name {
		case "Workshop":
			workshop = &data.Locations[i]
		case "Garage":
			garage = &data.Locations[i]
		}
	}
	require.NotNil(t, workshop)
	require.NotNil(t, garage)

	require.Equal(t, 15, workshop.InstanceCount)
	require.Equal(t, 10, workshop.DirectInstanceCount)
	require.Equal(t, 1, workshop.SubLocationCount)
	require.Len(t, workshop.Children, 1)
	require.Equal(t, "Cabinet", workshop.Children[0].Name)
	require.Equal(t, 5, workshop.Children[0].InstanceCount)
	require.Equal(t, 5, workshop.Children[0].DirectInstanceCount)

	require.Equal(t, 3, garage.InstanceCount)
	require.Equal(t, 3, garage.DirectInstanceCount)
	require.Equal(t, 0, garage.SubLocationCount)
}

func TestDashboardRootExcluded(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	rootID, _ := testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	for _, loc := range data.Locations {
		require.NotEqual(t, rootID, loc.ID)
	}
}

func TestDashboardRanking(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	_, _ = testutil.SeedRootLocation(t, db)

	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc_a', 'Workshop', NULL)`)
	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc_b', 'Garage', NULL)`)
	db.Exec(`INSERT INTO item_definitions (id, name) VALUES ('def1', 'Screw')`)
	db.Exec(`INSERT INTO item_instances (id, definition_id, quantity, location_id) VALUES ('i1', 'def1', 1, 'loc_a')`)
	db.Exec(`INSERT INTO item_instances (id, definition_id, quantity, location_id) VALUES ('i2', 'def1', 20, 'loc_b')`)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	require.Len(t, data.Locations, 2)

	require.Equal(t, "Garage", data.Locations[0].Name)
	require.Equal(t, 20, data.Locations[0].InstanceCount)

	require.Equal(t, "Workshop", data.Locations[1].Name)
	require.Equal(t, 1, data.Locations[1].InstanceCount)
}

func TestDashboardOnboardingAfterAdd(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	_, _ = testutil.SeedRootLocation(t, db)

	db.Exec(`INSERT INTO locations (id, name, parent_id) VALUES ('loc1', 'Closet', NULL)`)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	resp := doRequest(t, server.URL, "GET", "/api/v1/dashboard", "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var data dashboardDataResponse
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err)

	require.False(t, data.IsOnboarding)
}
