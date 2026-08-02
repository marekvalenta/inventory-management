-- +goose Up
-- +goose StatementBegin
UPDATE settings SET theme = 'dark' WHERE theme = 'system';
ALTER TABLE settings DROP COLUMN app_name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN app_name TEXT NOT NULL DEFAULT 'Inventory';
UPDATE settings SET theme = 'system' WHERE theme = 'dark';
-- +goose StatementEnd
