-- +goose Up
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    address TEXT NOT NULL,
    price BIGINT NOT NULL --в копейках
);

-- +goose Down
DROP TABLE IF EXISTS orders;
