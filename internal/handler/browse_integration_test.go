//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/marekvalenta/inventory-management/internal/testutil"
	"github.com/stretchr/testify/require"
)

type browseInstanceResponse struct {
	ID             string `json:"id"`
	DefinitionID   string `json:"definition_id"`
	DefinitionName string `json:"definition_name"`
	Quantity       int    `json:"quantity"`
	IsContainer    bool   `json:"is_container"`
	ChildCount     int    `json:"child_count"`
}

type browseNodeResponse struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Description       *string                  `json:"description"`
	Kind              string                   `json:"kind"`
	Children          []browseNodeResponse     `json:"children"`
	Instances         []browseInstanceResponse `json:"instances"`
	InstanceCount     int                      `json:"instance_count"`
	InstanceTruncated bool                     `json:"instance_truncated"`
}

func TestBrowse(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	rootID, _ := testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("returns empty instances for new root", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, "Home", nodes[0].Name)
		require.Equal(t, "location", nodes[0].Kind)
		require.Len(t, nodes[0].Instances, 0)
		require.Equal(t, 0, nodes[0].InstanceCount)
		require.False(t, nodes[0].InstanceTruncated)
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

	t.Run("creates definitions for browse instances", func(t *testing.T) {
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

	t.Run("browse shows instances at locations", func(t *testing.T) {
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
		require.Equal(t, 1, livingRoom.InstanceCount)
		require.False(t, livingRoom.InstanceTruncated)
		require.Len(t, livingRoom.Instances, 1)
		require.Equal(t, "Tool Box", livingRoom.Instances[0].DefinitionName)
		require.Equal(t, 2, livingRoom.Instances[0].Quantity)
		require.True(t, livingRoom.Instances[0].IsContainer)
		require.Equal(t, 0, livingRoom.Instances[0].ChildCount)
	})

	t.Run("browse shows container child count", func(t *testing.T) {
		var boxInstID string

		resp := doRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+boxDefID+`","quantity":1,"location_id":"`+livingRoomID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&inst)
		boxInstID = inst["id"].(string)

		resp = doRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+boxDefID+`","quantity":3,"parent_instance_id":"`+boxInstID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		resp = doRequest(t, server.URL, "GET", "/api/v1/browse", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var nodes []browseNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&nodes)
		require.NoError(t, err)

		var boxInst *browseInstanceResponse
		for i := range nodes[0].Children {
			if nodes[0].Children[i].Name == "Living Room" {
				for j := range nodes[0].Children[i].Instances {
					if nodes[0].Children[i].Instances[j].ChildCount > 0 {
						boxInst = &nodes[0].Children[i].Instances[j]
						break
					}
				}
			}
		}
		require.NotNil(t, boxInst)
		require.Equal(t, 1, boxInst.ChildCount)
		require.True(t, boxInst.IsContainer)
	})
}
