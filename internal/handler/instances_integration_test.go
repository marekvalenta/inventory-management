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

type instDetailResponse struct {
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
	FieldValues        []instFieldValueResp  `json:"field_values"`
	ChildInstanceCount int                   `json:"child_instance_count"`
	Breadcrumb         []breadcrumbEntryResp `json:"breadcrumb"`
	CreatedAt          string                `json:"created_at"`
	UpdatedAt          string                `json:"updated_at"`
}

type instFieldValueResp struct {
	FieldID    string           `json:"field_id"`
	FieldName  string           `json:"field_name"`
	FieldType  string           `json:"field_type"`
	EnumValues *json.RawMessage `json:"enum_values"`
	Value      *string          `json:"value"`
}

type breadcrumbEntryResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type instSummaryResp struct {
	ID                 string  `json:"id"`
	DefinitionID       string  `json:"definition_id"`
	DefinitionName     string  `json:"definition_name"`
	Quantity           int     `json:"quantity"`
	LocationID         *string `json:"location_id"`
	LocationName       *string `json:"location_name"`
	ParentInstanceID   *string `json:"parent_instance_id"`
	ParentInstanceName *string `json:"parent_instance_name"`
	UpdatedAt          string  `json:"updated_at"`
}

type instListResult struct {
	Instances  []instSummaryResp `json:"instances"`
	TotalCount int               `json:"total_count"`
	Truncated  bool              `json:"truncated,omitempty"`
}

type moveResultResp struct {
	Source *instDetailResponse `json:"source"`
	Target instDetailResponse  `json:"target"`
}

type instContentsResult struct {
	Instances []instSummaryResp `json:"instances"`
}

func doInstRequest(t *testing.T, serverURL, method, path string, body string) *http.Response {
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

func createTestLocation(t *testing.T, serverURL, name string) string {
	resp := doInstRequest(t, serverURL, "POST", "/api/v1/locations",
		`{"name":"`+name+`"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var loc struct {
		ID string `json:"id"`
	}
	err := json.NewDecoder(resp.Body).Decode(&loc)
	require.NoError(t, err)
	return loc.ID
}

func createTestDef(t *testing.T, serverURL, name, unit string, isContainer bool) string {
	containerStr := "false"
	if isContainer {
		containerStr = "true"
	}
	resp := doInstRequest(t, serverURL, "POST", "/api/v1/definitions",
		`{"name":"`+name+`","unit":"`+unit+`","is_container":`+containerStr+`,"fields":[],"tag_ids":[]}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var def struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&def)
	return def.ID
}

func createTestDefWithFields(t *testing.T, serverURL, name string, fields string) string {
	resp := doInstRequest(t, serverURL, "POST", "/api/v1/definitions",
		`{"name":"`+name+`","fields":`+fields+`,"tag_ids":[]}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var def struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&def)
	return def.ID
}

func TestInstancesCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	defID := createTestDef(t, server.URL, "Screw", "pcs", false)

	var createdID string

	t.Run("create instance at location", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var inst instDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&inst)
		require.NoError(t, err)

		require.NotEmpty(t, inst.ID)
		require.Equal(t, defID, inst.DefinitionID)
		require.Equal(t, "Screw", inst.DefinitionName)
		require.Equal(t, 5, inst.Quantity)
		require.Equal(t, "pcs", *inst.Unit)
		require.NotNil(t, inst.LocationID)
		require.Equal(t, locID, *inst.LocationID)
		require.NotNil(t, inst.LocationName)
		require.Equal(t, "Workshop", *inst.LocationName)
		require.Nil(t, inst.ParentInstanceID)
		require.NotEmpty(t, inst.Breadcrumb)
		require.Equal(t, "instance", inst.Breadcrumb[len(inst.Breadcrumb)-1].Kind)

		createdID = inst.ID
	})

	t.Run("get instance", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var inst instDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&inst)
		require.NoError(t, err)
		require.Equal(t, createdID, inst.ID)
		require.Equal(t, "Screw", inst.DefinitionName)
		require.Equal(t, 5, inst.Quantity)
	})

	t.Run("list instances", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result instListResult
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.Len(t, result.Instances, 1)
		require.Equal(t, 1, result.TotalCount)
		require.Equal(t, "Screw", result.Instances[0].DefinitionName)
	})

	t.Run("list instances filtered by location", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances?location_id="+locID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result instListResult
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.Len(t, result.Instances, 1)
	})

	t.Run("update instance quantity", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "PUT", "/api/v1/instances/"+createdID,
			`{"quantity":10}`)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var inst instDetailResponse
		err := json.NewDecoder(resp.Body).Decode(&inst)
		require.NoError(t, err)
		require.Equal(t, 10, inst.Quantity)
	})

	t.Run("get instance breadcrumb", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances/"+createdID+"/breadcrumb", "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var breadcrumb []breadcrumbEntryResp
		err := json.NewDecoder(resp.Body).Decode(&breadcrumb)
		require.NoError(t, err)
		require.NotEmpty(t, breadcrumb)
		require.Equal(t, "instance", breadcrumb[len(breadcrumb)-1].Kind)
	})

	t.Run("delete instance", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "DELETE", "/api/v1/instances/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("get after delete returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances/"+createdID, "")
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestInstancesErrors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	defID := createTestDef(t, server.URL, "Screw", "pcs", false)

	t.Run("create without XOR of location/container returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with both location and container returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`","parent_instance_id":"nonexistent"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with quantity 0 returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":0,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create with non-existent definition returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"nonexistent","quantity":1,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get non-existent returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances/nonexistent", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("update non-existent returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "PUT", "/api/v1/instances/nonexistent",
			`{"quantity":5}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("update quantity to 0 returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":1,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)

		resp = doInstRequest(t, server.URL, "PUT", "/api/v1/instances/"+inst.ID,
			`{"quantity":0}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("delete non-existent returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "DELETE", "/api/v1/instances/nonexistent", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("empty list returns empty array", func(t *testing.T) {
		db2 := testutil.SetupTestDB(t)
		testutil.RunMigrations(t, db2)
		testutil.SeedRootLocation(t, db2)

		server2 := testutil.NewTestServer(t, db2)
		defer server2.Close()

		resp := doInstRequest(t, server2.URL, "GET", "/api/v1/instances", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result instListResult
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.NotNil(t, result.Instances)
		require.Len(t, result.Instances, 0)
	})
}

func TestInstancesAutoMerge(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	defID := createTestDefWithFields(t, server.URL, "Screw", `[{"field_name":"Material","field_type":"text","is_required":false,"display_order":0}]`)

	t.Run("two creates with same properties merge", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst1 instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst1)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":3,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst2 instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst2)

		require.Equal(t, inst1.ID, inst2.ID)
		require.Equal(t, 8, inst2.Quantity)

		resp = doInstRequest(t, server.URL, "GET", "/api/v1/instances", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result instListResult
		json.NewDecoder(resp.Body).Decode(&result)
		require.Len(t, result.Instances, 1)
	})
}

func TestInstancesContainerNesting(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	boxDefID := createTestDef(t, server.URL, "Toolbox", "pcs", true)
	screwDefID := createTestDef(t, server.URL, "Screw", "pcs", false)

	var boxID string

	t.Run("create container instance", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+boxDefID+`","quantity":1,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)
		boxID = inst.ID
	})

	t.Run("create instance inside container", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":50,"parent_instance_id":"`+boxID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)
		require.NotNil(t, inst.ParentInstanceID)
		require.Equal(t, boxID, *inst.ParentInstanceID)
		require.Equal(t, "Toolbox", *inst.ParentInstanceName)
	})

	t.Run("create inside non-container returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":1,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var screwInst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&screwInst)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":1,"parent_instance_id":"`+screwInst.ID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("container contents", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "GET", "/api/v1/instances/"+boxID+"/contents", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var contents instContentsResult
		json.NewDecoder(resp.Body).Decode(&contents)
		require.Len(t, contents.Instances, 1)
		require.Equal(t, "Screw", contents.Instances[0].DefinitionName)
	})

	t.Run("breadcrumb for nested instance includes container", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":20,"parent_instance_id":"`+boxID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)

		resp = doInstRequest(t, server.URL, "GET", "/api/v1/instances/"+inst.ID+"/breadcrumb", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var breadcrumb []breadcrumbEntryResp
		json.NewDecoder(resp.Body).Decode(&breadcrumb)
		require.NotEmpty(t, breadcrumb)

		hasLocation := false
		hasInstance := false
		for _, b := range breadcrumb {
			if b.Kind == "location" {
				hasLocation = true
			}
			if b.Kind == "instance" {
				hasInstance = true
			}
		}
		require.True(t, hasLocation)
		require.True(t, hasInstance)
	})
}

func TestInstancesMove(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locA := createTestLocation(t, server.URL, "Workshop A")
	locB := createTestLocation(t, server.URL, "Workshop B")
	screwDefID := createTestDefWithFields(t, server.URL, "Screw", `[{"field_name":"Material","field_type":"text","is_required":false,"display_order":0}]`)

	resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
		`{"definition_id":"`+screwDefID+`","quantity":10,"location_id":"`+locA+`"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var sourceInst instDetailResponse
	json.NewDecoder(resp.Body).Decode(&sourceInst)

	t.Run("partial move to different location", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+sourceInst.ID+"/move",
			`{"quantity":3,"target_location_id":"`+locB+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result moveResultResp
		json.NewDecoder(resp.Body).Decode(&result)
		require.NotNil(t, result.Source)
		require.Equal(t, 7, result.Source.Quantity)
		require.Equal(t, 3, result.Target.Quantity)
		require.NotNil(t, result.Target.LocationID)
		require.Equal(t, locB, *result.Target.LocationID)
	})

	t.Run("full move deletes source", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+sourceInst.ID+"/move",
			`{"quantity":7,"target_location_id":"`+locB+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result moveResultResp
		json.NewDecoder(resp.Body).Decode(&result)
		require.Nil(t, result.Source)
		require.NotEmpty(t, result.Target.ID)
	})

	t.Run("move to same location returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":5,"location_id":"`+locA+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+inst.ID+"/move",
			`{"quantity":1,"target_location_id":"`+locA+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("move quantity exceeds available returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":2,"location_id":"`+locA+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+inst.ID+"/move",
			`{"quantity":10,"target_location_id":"`+locB+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("move to non-existent location returns 404", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":1,"location_id":"`+locA+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var inst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&inst)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+inst.ID+"/move",
			`{"quantity":1,"target_location_id":"nonexistent"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestInstancesMoveContainer(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	boxDefID := createTestDef(t, server.URL, "Box", "pcs", true)
	screwDefID := createTestDef(t, server.URL, "Screw", "pcs", false)

	resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
		`{"definition_id":"`+boxDefID+`","quantity":1,"location_id":"`+locID+`"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var boxInst instDetailResponse
	json.NewDecoder(resp.Body).Decode(&boxInst)

	t.Run("move screws into box", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+screwDefID+`","quantity":100,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var screwInst instDetailResponse
		json.NewDecoder(resp.Body).Decode(&screwInst)

		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+screwInst.ID+"/move",
			`{"quantity":50,"target_parent_instance_id":"`+boxInst.ID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var result moveResultResp
		json.NewDecoder(resp.Body).Decode(&result)
		require.NotNil(t, result.Source)
		require.NotNil(t, result.Target.ParentInstanceID)
		require.Equal(t, boxInst.ID, *result.Target.ParentInstanceID)
	})

	t.Run("cannot move to self", func(t *testing.T) {
		resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances/"+boxInst.ID+"/move",
			`{"quantity":1,"target_parent_instance_id":"`+boxInst.ID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestInstancesDeleteGuard(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	boxDefID := createTestDef(t, server.URL, "Box", "pcs", true)
	screwDefID := createTestDef(t, server.URL, "Screw", "pcs", false)

	resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
		`{"definition_id":"`+boxDefID+`","quantity":1,"location_id":"`+locID+`"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var boxInst instDetailResponse
	json.NewDecoder(resp.Body).Decode(&boxInst)

	resp = doInstRequest(t, server.URL, "POST", "/api/v1/instances",
		`{"definition_id":"`+screwDefID+`","quantity":10,"parent_instance_id":"`+boxInst.ID+`"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	t.Run("cannot delete container with children", func(t *testing.T) {
		resp = doInstRequest(t, server.URL, "DELETE", "/api/v1/instances/"+boxInst.ID, "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusConflict, resp.StatusCode)

		var errResp errorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		require.Equal(t, "instance_has_children", errResp.Code)
	})
}

func TestInstancesFieldValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	testutil.RunMigrations(t, db)
	testutil.SeedRootLocation(t, db)

	server := testutil.NewTestServer(t, db)
	defer server.Close()

	locID := createTestLocation(t, server.URL, "Workshop")
	defID := createTestDefWithFields(t, server.URL, "Screw", `[
		{"field_name":"Material","field_type":"enum","enum_values":["Steel","Brass"],"is_required":true,"display_order":0},
		{"field_name":"Length","field_type":"number","is_required":false,"display_order":1},
		{"field_name":"Coated","field_type":"boolean","is_required":false,"display_order":2}
	]`)

	t.Run("required field missing returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`"}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid enum value returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`","field_values":[{"field_id":"nonexistent","value":"Steel"}]}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid number value returns 400", func(t *testing.T) {
		resp := doInstRequest(t, server.URL, "POST", "/api/v1/instances",
			`{"definition_id":"`+defID+`","quantity":5,"location_id":"`+locID+`","field_values":[{"field_id":"nonexistent","value":"Steel"}]}`)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
