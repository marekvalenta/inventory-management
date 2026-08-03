//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/marekvalenta/inventory-management/internal/testutil"
	"github.com/stretchr/testify/require"
)

type browseStackResponse struct {
	DefinitionID   string  `json:"definition_id"`
	DefinitionName string  `json:"definition_name"`
	Unit           *string `json:"unit"`
	TotalQuantity  int     `json:"total_quantity"`
	InstanceCount  int     `json:"instance_count"`
	IsContainer    bool    `json:"is_container"`
	ChildCount     int     `json:"child_count"`
}

type browseNodeResponse struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    *string                `json:"description"`
	Kind           string                 `json:"kind"`
	Children       []browseNodeResponse   `json:"children"`
	Stacks         []browseStackResponse  `json:"stacks"`
	StackCount     int                    `json:"stack_count"`
	StackTruncated bool                   `json:"stack_truncated"`
}

func TestBrowse(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	rootID, _ := testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("returns empty stacks for new root", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "Home", nodes[0].Name)
		require.Equal(t, "location", nodes[0].Kind)
		require.Len(t, nodes[0].Stacks, 0)
		require.Equal(t, 0, nodes[0].StackCount)
		require.False(t, nodes[0].StackTruncated)
	})

	var livingRoomID string

	t.Run("creates sub-locations for browse tree", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Living Room","parent_id":"`+rootID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var loc locationResponse
		json.NewDecoder(resp.Body).Decode(&loc)
		livingRoomID = loc.ID

		resp = doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Attic","parent_id":"`+rootID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("browse shows location hierarchy", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "Home", nodes[0].Name)
		require.Len(t, nodes[0].Children, 2)
		require.Equal(t, "Attic", nodes[0].Children[0].Name)
		require.Equal(t, "Living Room", nodes[0].Children[1].Name)
		require.Equal(t, "location", nodes[0].Children[0].Kind)
	})

	var boxDefID string

	t.Run("creates definitions for browse stacks", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Tool Box","unit":"pcs","is_container":true}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var def map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&def)
		boxDefID = def["id"].(string)

		resp = doRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Hammer","unit":"pcs","is_container":false}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("browse shows stacks at locations", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+boxDefID+`","quantity":2,"location_id":"`+livingRoomID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)

		homeChildren := nodes[0].Children
		require.Len(t, homeChildren, 2)

		var livingRoom *browseNodeResponse
		for i := range homeChildren {
			if homeChildren[i].Name == "Living Room" {
				livingRoom = &homeChildren[i]
				break
			}
		}
		require.NotNil(t, livingRoom)
		require.Equal(t, 1, livingRoom.StackCount)
		require.False(t, livingRoom.StackTruncated)
		require.Len(t, livingRoom.Stacks, 1)
		require.Equal(t, "Tool Box", livingRoom.Stacks[0].DefinitionName)
		require.Equal(t, 2, livingRoom.Stacks[0].TotalQuantity)
		require.Equal(t, 1, livingRoom.Stacks[0].InstanceCount)
		require.True(t, livingRoom.Stacks[0].IsContainer)
	})

	t.Run("browse shows stacks grouped by definition", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+boxDefID+`","quantity":1,"location_id":"`+livingRoomID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)

		var livingRoom *browseNodeResponse
		for i := range nodes[0].Children {
			if nodes[0].Children[i].Name == "Living Room" {
				livingRoom = &nodes[0].Children[i]
				break
			}
		}
		require.NotNil(t, livingRoom)
		require.Equal(t, 1, livingRoom.StackCount)
		require.Len(t, livingRoom.Stacks, 1)
		require.Equal(t, "Tool Box", livingRoom.Stacks[0].DefinitionName)
		require.Equal(t, 3, livingRoom.Stacks[0].TotalQuantity)
		require.Equal(t, 1, livingRoom.Stacks[0].InstanceCount)
	})
}
