-- +goose Up
-- +goose StatementBegin
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
    app_name         TEXT NOT NULL DEFAULT 'Inventory',
    theme            TEXT NOT NULL DEFAULT 'system',
    root_location_id TEXT REFERENCES locations(id),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS instance_field_values;
DROP TABLE IF EXISTS definition_field_overrides;
DROP TABLE IF EXISTS item_instances;
DROP TABLE IF EXISTS definition_fields;
DROP TABLE IF EXISTS item_definitions;
DROP TABLE IF EXISTS definition_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS locations;
-- +goose StatementEnd
