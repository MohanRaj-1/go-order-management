-- +goose Up
CREATE TABLE inventory (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL,
    physical_quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT inventory_product_fk
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE RESTRICT,

    CONSTRAINT inventory_product_unique
        UNIQUE (product_id),

    CONSTRAINT inventory_physical_quantity_nonnegative
        CHECK (physical_quantity >= 0),

    CONSTRAINT inventory_reserved_quantity_nonnegative
        CHECK (reserved_quantity >= 0),

    CONSTRAINT inventory_reserved_not_exceed_physical
        CHECK (reserved_quantity <= physical_quantity)
);


-- +goose Down
DROP TABLE inventory;
