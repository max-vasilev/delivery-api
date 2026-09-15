-- +goose Up
CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK(quantity > 0),
    price BIGINT NOT NULL CHECK(price >= 0), -- в копейках
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- +goose Down
DROP TABLE IF EXISTS order_items;
