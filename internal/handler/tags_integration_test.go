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

type tagResponse struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	Color                 *string `json:"color"`
	LinkedDefinitionsCount int    `json:"linked_definitions_count"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
}

type deleteTagResponse struct {
	Deleted                bool `json:"deleted"`
	LinkedDefinitionsCount int  `json:"linked_definitions_count"`
}

func doTagRequest(t *testing.T, serverURL, method, path string, body string) *http.Response {
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

func TestTagsCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	var createdID string

	t.Run("create tag", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"Fasteners","color":"#E8A838"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var tag tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tag)
		require.NoError(t, err)

		require.NotEmpty(t, tag.ID)
		require.Equal(t, "Fasteners", tag.Name)
		require.NotNil(t, tag.Color)
		require.Equal(t, "#E8A838", *tag.Color)
		require.Equal(t, 0, tag.LinkedDefinitionsCount)
		require.NotEmpty(t, tag.CreatedAt)
		require.NotEmpty(t, tag.UpdatedAt)

		createdID = tag.ID
	})

	t.Run("get tag", func(t *testing.T) {
		require.NotEmpty(t, createdID)

		resp := doTagRequest(t, server.URL, "GET", "/api/v1/tags/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tag tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tag)
		require.NoError(t, err)

		require.Equal(t, createdID, tag.ID)
		require.Equal(t, "Fasteners", tag.Name)
	})

	t.Run("list tags", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "GET", "/api/v1/tags", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tags)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, "Fasteners", tags[0].Name)
	})

	t.Run("create tag without color", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"Office"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var tag tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tag)
		require.NoError(t, err)
		require.Equal(t, "Office", tag.Name)
		require.Nil(t, tag.Color)
		require.Equal(t, 0, tag.LinkedDefinitionsCount)
	})

	t.Run("list tags sorted by name", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "GET", "/api/v1/tags", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tags)
		require.NoError(t, err)
		require.Len(t, tags, 2)
		require.Equal(t, "Fasteners", tags[0].Name)
		require.Equal(t, "Office", tags[1].Name)
	})

	t.Run("update tag name", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "PUT", "/api/v1/tags/"+createdID,
			`{"name":"Hardware"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tag tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tag)
		require.NoError(t, err)
		require.Equal(t, "Hardware", tag.Name)
		require.NotNil(t, tag.Color)
		require.Equal(t, "#E8A838", *tag.Color)
	})

	t.Run("update tag color", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "PUT", "/api/v1/tags/"+createdID,
			`{"color":"#6B8E5A"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tag tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tag)
		require.NoError(t, err)
		require.Equal(t, "Hardware", tag.Name)
		require.NotNil(t, tag.Color)
		require.Equal(t, "#6B8E5A", *tag.Color)
	})

	t.Run("delete tag", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "DELETE", "/api/v1/tags/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var delResp deleteTagResponse
		err := json.NewDecoder(resp.Body).Decode(&delResp)
		require.NoError(t, err)
		require.True(t, delResp.Deleted)
		require.Equal(t, 0, delResp.LinkedDefinitionsCount)
	})

	t.Run("get after delete returns 404", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "GET", "/api/v1/tags/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestTagsErrors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("get non-existent returns 404", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "GET", "/api/v1/tags/nonexistent", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("create with empty name returns 400", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":""}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Equal(t, "validation_failed", errResp.Code)
	})

	t.Run("create with name too short returns 400", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"A"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with name too long returns 400", func(t *testing.T) {
		longName := strings.Repeat("x", 101)
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"`+longName+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with duplicate name returns 409", func(t *testing.T) {
		doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"DuplicateMe"}`)

		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"DuplicateMe"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Equal(t, "duplicate_name", errResp.Code)
	})

	t.Run("update non-existent returns 404", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "PUT", "/api/v1/tags/nonexistent",
			`{"name":"Test"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("update to duplicate name returns 409", func(t *testing.T) {
		doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"First"}`)

		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"Second"}`)
		defer resp.Body.Close()
		var second tagResponse
		json.NewDecoder(resp.Body).Decode(&second)

		resp = doTagRequest(t, server.URL, "PUT", "/api/v1/tags/"+second.ID,
			`{"name":"First"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Equal(t, "duplicate_name", errResp.Code)
	})

	t.Run("update to same name succeeds", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"SameName"}`)
		defer resp.Body.Close()
		var tag tagResponse
		json.NewDecoder(resp.Body).Decode(&tag)

		resp = doTagRequest(t, server.URL, "PUT", "/api/v1/tags/"+tag.ID,
			`{"name":"SameName"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "DELETE", "/api/v1/tags/nonexistent", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("create with color too long returns 400", func(t *testing.T) {
		resp := doTagRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"ValidName","color":"12345678901"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		db2 := testutil.SetupTestDB(t)
		testutil.RunMigrations(t, db2)

		server2 := testutil.NewTestServer(t, db2)
		defer server2.Close()

		resp := doTagRequest(t, server2.URL, "GET", "/api/v1/tags", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var tags []tagResponse
		err := json.NewDecoder(resp.Body).Decode(&tags)
		require.NoError(t, err)
		require.NotNil(t, tags)
		require.Len(t, tags, 0)
	})
}
