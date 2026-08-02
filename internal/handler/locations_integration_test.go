//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/marekvalenta/inventory-management/internal/testutil"
	"github.com/stretchr/testify/require"
)

type locationResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type treeNodeResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Children    []treeNodeResponse `json:"children"`
}

type breadcrumbResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type contentsResponse struct {
	SubLocations []locationResponse `json:"sub_locations"`
	Instances    []interface{}       `json:"instances"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func doRequest(t *testing.T, serverURL, method, path string, body string) *http.Response {
	t.Helper()

	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}

	req, err := http.NewRequest(method, serverURL+path, reqBody)
	require.NoError(t, err)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func TestLocationsCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	_, _ = testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	var createdID string

	t.Run("create location", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Living Room","description":"Main living area"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var loc locationResponse
		err := json.NewDecoder(resp.Body).Decode(&loc)
		require.NoError(t, err)

		require.NotEmpty(t, loc.ID)
		require.Equal(t, "Living Room", loc.Name)
		require.Equal(t, "Main living area", *loc.Description)
		require.Nil(t, loc.ParentID)
		require.NotEmpty(t, loc.CreatedAt)
		require.NotEmpty(t, loc.UpdatedAt)

		createdID = loc.ID
	})

	t.Run("get location", func(t *testing.T) {
		require.NotEmpty(t, createdID)

		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var loc locationResponse
		err := json.NewDecoder(resp.Body).Decode(&loc)
		require.NoError(t, err)

		require.Equal(t, createdID, loc.ID)
		require.Equal(t, "Living Room", loc.Name)
	})

	t.Run("create child location", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Bookshelf","parent_id":"`+createdID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var loc locationResponse
		err := json.NewDecoder(resp.Body).Decode(&loc)
		require.NoError(t, err)

		require.NotNil(t, loc.ParentID)
		require.Equal(t, createdID, *loc.ParentID)
	})

	t.Run("list all locations", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var locations []locationResponse
		err := json.NewDecoder(resp.Body).Decode(&locations)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(locations), 2)
	})

	t.Run("list locations filtered by parent", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations?parent_id="+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var locations []locationResponse
		err := json.NewDecoder(resp.Body).Decode(&locations)
		require.NoError(t, err)
		require.Len(t, locations, 1)
		require.Equal(t, "Bookshelf", locations[0].Name)
	})

	t.Run("get children", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/"+createdID+"/children", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var children []locationResponse
		err := json.NewDecoder(resp.Body).Decode(&children)
		require.NoError(t, err)
		require.Len(t, children, 1)
		require.Equal(t, "Bookshelf", children[0].Name)
	})

	t.Run("get breadcrumb", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/"+createdID+"/breadcrumb", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var breadcrumb []breadcrumbResponse
		err := json.NewDecoder(resp.Body).Decode(&breadcrumb)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(breadcrumb), 1)
		require.Equal(t, "Living Room", breadcrumb[len(breadcrumb)-1].Name)
	})

	t.Run("get contents", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/"+createdID+"/contents", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var contents contentsResponse
		err := json.NewDecoder(resp.Body).Decode(&contents)
		require.NoError(t, err)
		require.Len(t, contents.SubLocations, 1)
		require.Len(t, contents.Instances, 0)
	})

	t.Run("get tree", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/tree", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tree []treeNodeResponse
		err := json.NewDecoder(resp.Body).Decode(&tree)
		require.NoError(t, err)
		require.NotEmpty(t, tree)
	})

	t.Run("update location", func(t *testing.T) {
		resp := doRequest(t, server.URL, "PUT", "/api/v1/locations/"+createdID,
			`{"name":"Updated Room"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var loc locationResponse
		err := json.NewDecoder(resp.Body).Decode(&loc)
		require.NoError(t, err)
		require.Equal(t, "Updated Room", loc.Name)
	})

	t.Run("delete child", func(t *testing.T) {
		var childID string

		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Temp","parent_id":"`+createdID+`"}`)
		defer resp.Body.Close()
		var loc locationResponse
		json.NewDecoder(resp.Body).Decode(&loc)
		childID = loc.ID

		resp = doRequest(t, server.URL, "DELETE", "/api/v1/locations/"+childID, "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusNoContent, resp.StatusCode)

		resp = doRequest(t, server.URL, "GET", "/api/v1/locations/"+childID, "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get non-existent returns 404", func(t *testing.T) {
		resp := doRequest(t, server.URL, "GET", "/api/v1/locations/nonexistent", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("create with invalid name returns 400", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"A"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with non-existent parent returns 400", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Orphan","parent_id":"nonexistent"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("delete location with children returns 409", func(t *testing.T) {
		resp := doRequest(t, server.URL, "DELETE", "/api/v1/locations/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Equal(t, "location_not_empty", errResp.Code)
	})

	t.Run("cannot delete root location", func(t *testing.T) {
		rootID := "00000000-0000-0000-0000-000000000001"

		resp := doRequest(t, server.URL, "DELETE", "/api/v1/locations/"+rootID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cannot reparent root location", func(t *testing.T) {
		rootID := "00000000-0000-0000-0000-000000000001"

		resp := doRequest(t, server.URL, "PUT", "/api/v1/locations/"+rootID,
			`{"parent_id":"`+createdID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cycle detection", func(t *testing.T) {
		resp := doRequest(t, server.URL, "POST", "/api/v1/locations",
			`{"name":"Parent","parent_id":"`+createdID+`"}`)
		defer resp.Body.Close()
		var parentLoc locationResponse
		json.NewDecoder(resp.Body).Decode(&parentLoc)

		resp = doRequest(t, server.URL, "PUT", "/api/v1/locations/"+createdID,
			`{"parent_id":"`+parentLoc.ID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
