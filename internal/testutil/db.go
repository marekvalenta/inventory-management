package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()

	db, err := sql.Open("sqlite", f.Name()+"?_journal_mode=WAL&_foreign_keys=ON")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		db.Close()
		os.Remove(f.Name())
	})

	return db
}

func RunMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	schema := `
CREATE TABLE locations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    parent_id   TEXT REFERENCES locations(id) ON DELETE RESTRICT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tags (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE item_definitions (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    description     TEXT,
    parent_def_id   TEXT REFERENCES item_definitions(id) ON DELETE RESTRICT,
    unit            TEXT,
    is_container    BOOLEAN NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE definition_tags (
    definition_id TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    tag_id        TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (definition_id, tag_id)
);

CREATE TABLE definition_fields (
    id              TEXT PRIMARY KEY,
    definition_id   TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    field_name      TEXT NOT NULL,
    field_type      TEXT NOT NULL,
    enum_values     TEXT,
    is_required     BOOLEAN NOT NULL DEFAULT 0,
    display_order   INTEGER NOT NULL DEFAULT 0,
    default_value   TEXT,
    is_child_editable BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE item_instances (
    id                  TEXT PRIMARY KEY,
    definition_id       TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE RESTRICT,
    quantity            INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    location_id         TEXT REFERENCES locations(id),
    parent_instance_id  TEXT REFERENCES item_instances(id) ON DELETE RESTRICT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_single_parent CHECK (
        (location_id IS NULL) != (parent_instance_id IS NULL)
    )
);

CREATE TABLE definition_field_overrides (
    definition_id   TEXT NOT NULL REFERENCES item_definitions(id) ON DELETE CASCADE,
    parent_field_id TEXT NOT NULL REFERENCES definition_fields(id) ON DELETE CASCADE,
    default_value   TEXT,
    PRIMARY KEY (definition_id, parent_field_id)
);

CREATE TABLE instance_field_values (
    id          TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL REFERENCES item_instances(id) ON DELETE CASCADE,
    field_id    TEXT NOT NULL REFERENCES definition_fields(id),
    value       TEXT,
    UNIQUE (instance_id, field_id)
);

CREATE TABLE settings (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    theme            TEXT NOT NULL DEFAULT 'dark',
    root_location_id TEXT REFERENCES locations(id),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	_, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_item_definitions_name ON item_definitions(name)`)
	if err != nil {
		t.Fatalf("failed to create unique index on item_definitions.name: %v", err)
	}
}

func SeedRootLocation(t *testing.T, db *sql.DB) (rootID string, settingsID string) {
	t.Helper()

	rootID = "00000000-0000-0000-0000-000000000001"

	_, err := db.Exec(
		"INSERT INTO locations (id, name, description) VALUES (?, ?, ?)",
		rootID, "Home", "Root location",
	)
	if err != nil {
		t.Fatalf("failed to seed root location: %v", err)
	}

	_, err = db.Exec(
		"INSERT INTO settings (id, theme, root_location_id) VALUES (1, ?, ?)",
		"dark", rootID,
	)
	if err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	return rootID, rootID
}
