package testutil

import (
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/marekvalenta/inventory-management/internal/router"
)

func NewTestServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()

	r := router.NewTestRouter(db)
	return httptest.NewServer(r)
}
