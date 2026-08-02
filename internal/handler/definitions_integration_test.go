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

type defSummaryResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	ParentDefID     *string         `json:"parent_def_id"`
	ParentDefName   *string         `json:"parent_def_name"`
	Unit            *string         `json:"unit"`
	IsContainer     bool            `json:"is_container"`
	TotalInstances  int             `json:"total_instances"`
	Tags            []tagResponse   `json:"tags"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type defDetailResponse struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         *string                `json:"description"`
	ParentDefID         *string                `json:"parent_def_id"`
	ParentDefName       *string                `json:"parent_def_name"`
	Unit                *string                `json:"unit"`
	IsContainer         bool                   `json:"is_container"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
	Fields              []defFieldResponse     `json:"fields"`
	Tags                []tagResponse          `json:"tags"`
	InstancesSummary    instancesSummaryResp   `json:"instances_summary"`
	ChildDefinitionCount int                   `json:"child_definition_count"`
}

type defFieldResponse struct {
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

type instancesSummaryResp struct {
	TotalInstances   int                       `json:"total_instances"`
	TotalQuantity    int                       `json:"total_quantity"`
	ByLocation       []locationInstanceCount   `json:"by_location"`
	ByParentInstance []parentInstanceCount     `json:"by_parent_instance"`
}

type locationInstanceCount struct {
	LocationID    string `json:"location_id"`
	LocationName  string `json:"location_name"`
	InstanceCount int    `json:"instance_count"`
	TotalQuantity int    `json:"total_quantity"`
}

type parentInstanceCount struct {
	ParentInstanceID   string `json:"parent_instance_id"`
	ParentInstanceName string `json:"parent_instance_name"`
	LocationID         string `json:"location_id"`
	LocationName       string `json:"location_name"`
	InstanceCount      int    `json:"instance_count"`
	TotalQuantity      int    `json:"total_quantity"`
}

type overrideResponse struct {
	DefinitionID  string  `json:"definition_id"`
	ParentFieldID string  `json:"parent_field_id"`
	DefaultValue  *string `json:"default_value"`
}

type overridesBodyResponse struct {
	Overrides []overrideResponse `json:"overrides"`
}

func doDefRequest(t *testing.T, serverURL, method, path string, body string) *http.Response {
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

func TestDefinitionsCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	var createdID string

	t.Run("create definition", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Screw","description":"Various screws","unit":"pcs","is_container":false,"fields":[],"tag_ids":[]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)

		require.NotEmpty(t, def.ID)
		require.Equal(t, "Screw", def.Name)
		require.Equal(t, "pcs", *def.Unit)
		require.Equal(t, 0, len(def.Fields))
		require.Equal(t, 0, len(def.Tags))
		require.Equal(t, 0, def.InstancesSummary.TotalInstances)
		require.Equal(t, 0, def.ChildDefinitionCount)

		createdID = def.ID
	})

	t.Run("get definition", func(t *testing.T) {
		require.NotEmpty(t, createdID)

		resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Equal(t, createdID, def.ID)
		require.Equal(t, "Screw", def.Name)
	})

	t.Run("list definitions", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var defs []defSummaryResponse
		err := json.NewDecoder(resp.Body).Decode(&defs)
		require.NoError(t, err)
		require.Len(t, defs, 1)
		require.Equal(t, "Screw", defs[0].Name)
	})

	t.Run("create definition with fields and tags", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/tags",
			`{"name":"Fasteners","color":"#E8A838"}`)
		resp.Body.Close()

		resp = doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Bolt","fields":[{"field_name":"Material","field_type":"enum","enum_values":["Steel","Brass"],"is_required":true,"display_order":0,"is_child_editable":true}],"tag_ids":[]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Equal(t, "Bolt", def.Name)
		require.Len(t, def.Fields, 1)
		require.Equal(t, "Material", def.Fields[0].FieldName)
		require.Equal(t, "enum", def.Fields[0].FieldType)
		require.NotNil(t, def.Fields[0].EnumValues)
		require.True(t, def.Fields[0].IsChildEditable)
		require.Nil(t, def.Fields[0].InheritedFromDefID)
	})

	t.Run("update definition name", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+createdID,
			`{"name":"Wood Screw"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Equal(t, "Wood Screw", def.Name)
	})

	t.Run("update definition fields", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+createdID,
			`{"fields":[{"field_name":"Length","field_type":"number","is_required":true,"display_order":0}]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Len(t, def.Fields, 1)
		require.Equal(t, "Length", def.Fields[0].FieldName)
	})

	t.Run("delete definition", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "DELETE", "/api/v1/definitions/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("get after delete returns 404", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestDefinitionsErrors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("get non-existent returns 404", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions/nonexistent", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("create with empty name returns 400", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"A"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with duplicate name returns 409", func(t *testing.T) {
		doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"UniqueScrew"}`)

		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"UniqueScrew"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Equal(t, "duplicate_name", errResp.Code)
	})

	t.Run("create with non-existent parent returns 400", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"ChildScrew","parent_def_id":"nonexistent"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("update non-existent returns 404", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/nonexistent",
			`{"name":"Test"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "DELETE", "/api/v1/definitions/nonexistent", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		db2 := testutil.SetupTestDB(t)
		testutil.RunMigrations(t, db2)

		server2 := testutil.NewTestServer(t, db2)
		defer server2.Close()

		resp := doDefRequest(t, server2.URL, "GET", "/api/v1/definitions", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var defs []defSummaryResponse
		err := json.NewDecoder(resp.Body).Decode(&defs)
		require.NoError(t, err)
		require.NotNil(t, defs)
		require.Len(t, defs, 0)
	})
}

func TestDefinitionsInheritance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	var parentID, childID string

	t.Run("create parent with fields", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Fastener","fields":[{"field_name":"Material","field_type":"enum","enum_values":["Steel","Brass"],"is_required":true,"display_order":0,"default_value":"Steel","is_child_editable":true},{"field_name":"Weight","field_type":"number","is_required":false,"display_order":1,"is_child_editable":false}]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Len(t, def.Fields, 2)
		require.Equal(t, "Material", def.Fields[0].FieldName)
		require.Equal(t, "Weight", def.Fields[1].FieldName)

		parentID = def.ID
	})

	t.Run("create child definition inheriting fields", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Screw","parent_def_id":"`+parentID+`","fields":[{"field_name":"Length","field_type":"number","is_required":true,"display_order":0}]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var def defDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&def)
		require.NoError(t, err)
		require.Equal(t, parentID, *def.ParentDefID)
		require.Equal(t, "Fastener", *def.ParentDefName)

		require.Len(t, def.Fields, 3)

		require.Equal(t, "Length", def.Fields[0].FieldName)
		require.Nil(t, def.Fields[0].InheritedFromDefID)

		require.Equal(t, "Material", def.Fields[1].FieldName)
		require.NotNil(t, def.Fields[1].InheritedFromDefID)
		require.Equal(t, parentID, *def.Fields[1].InheritedFromDefID)
		require.Equal(t, "Steel", *def.Fields[1].DefaultValue)

		require.Equal(t, "Weight", def.Fields[2].FieldName)
		require.NotNil(t, def.Fields[2].InheritedFromDefID)

		childID = def.ID
	})

	t.Run("override inherited field default", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions/"+parentID, "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var parentDef defDetailResponse
		json.NewDecoder(resp.Body).Decode(&parentDef)

		var matFieldID string
		for _, f := range parentDef.Fields {
			if f.FieldName == "Material" {
				matFieldID = f.ID
			}
		}
		require.NotEmpty(t, matFieldID)

		resp = doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+childID+"/overrides",
			`{"overrides":[{"parent_field_id":"`+matFieldID+`","default_value":"Brass"}]}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body overridesBodyResponse
		err := json.NewDecoder(resp.Body).Decode(&body)
		require.NoError(t, err)
		require.Len(t, body.Overrides, 1)
		require.Equal(t, "Brass", *body.Overrides[0].DefaultValue)
	})

	t.Run("cannot delete parent with children", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "DELETE", "/api/v1/definitions/"+parentID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		require.Contains(t, errResp.Error, "child definitions")
	})
}

func TestDefinitionsOverrides(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("create parent and child for overrides", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"BaseDef","fields":[{"field_name":"Color","field_type":"enum","enum_values":["Red","Blue"],"is_required":false,"display_order":0,"default_value":"Red","is_child_editable":true},{"field_name":"SealedField","field_type":"text","is_required":false,"display_order":1,"default_value":"locked","is_child_editable":false}]}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var parent defDetailResponse
		json.NewDecoder(resp.Body).Decode(&parent)

		var colorFieldID string
		var sealedFieldID string
		for _, f := range parent.Fields {
			if f.FieldName == "Color" {
				colorFieldID = f.ID
			}
			if f.FieldName == "SealedField" {
				sealedFieldID = f.ID
			}
		}

		resp = doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"ChildDef","parent_def_id":"`+parent.ID+`","fields":[]}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var child defDetailResponse
		json.NewDecoder(resp.Body).Decode(&child)

		t.Run("override editable field", func(t *testing.T) {
			resp := doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+child.ID+"/overrides",
				`{"overrides":[{"parent_field_id":"`+colorFieldID+`","default_value":"Blue"}]}`)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			var body overridesBodyResponse
			err := json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)
			require.Len(t, body.Overrides, 1)
			require.Equal(t, "Blue", *body.Overrides[0].DefaultValue)
		})

		t.Run("override sealed field returns 400", func(t *testing.T) {
			resp := doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+child.ID+"/overrides",
				`{"overrides":[{"parent_field_id":"`+sealedFieldID+`","default_value":"unlocked"}]}`)
			defer resp.Body.Close()

			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})

		t.Run("get child shows overridden default", func(t *testing.T) {
			resp := doDefRequest(t, server.URL, "GET", "/api/v1/definitions/"+child.ID, "")
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			var def defDetailResponse
			json.NewDecoder(resp.Body).Decode(&def)

			for _, f := range def.Fields {
				if f.FieldName == "Color" {
					require.Equal(t, "Blue", *f.DefaultValue)
				}
			}
		})
	})
}

func TestDefinitionsSelfParent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("cannot set parent to self", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"SelfTest"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var def defDetailResponse
		json.NewDecoder(resp.Body).Decode(&def)

		resp = doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+def.ID,
			`{"parent_def_id":"`+def.ID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestDefinitionsCycleDetection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	t.Run("detect cycle", func(t *testing.T) {
		resp := doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Grandparent"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var gp defDetailResponse
		json.NewDecoder(resp.Body).Decode(&gp)

		resp = doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Parent","parent_def_id":"`+gp.ID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var p defDetailResponse
		json.NewDecoder(resp.Body).Decode(&p)

		resp = doDefRequest(t, server.URL, "POST", "/api/v1/definitions",
			`{"name":"Child","parent_def_id":"`+p.ID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var c defDetailResponse
		json.NewDecoder(resp.Body).Decode(&c)

		resp = doDefRequest(t, server.URL, "PUT", "/api/v1/definitions/"+gp.ID,
			`{"parent_def_id":"`+c.ID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
