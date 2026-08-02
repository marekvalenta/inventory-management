-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_item_definitions_name ON item_definitions(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_item_definitions_name;
-- +goose StatementEnd
