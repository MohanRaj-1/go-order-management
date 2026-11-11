-- +goose Up
CREATE TABLE products (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    price NUMERIC(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT products_name_not_empty
        CHECK (LENGTH(TRIM(name)) > 0),

    CONSTRAINT products_sku_not_empty
        CHECK (LENGTH(TRIM(sku)) > 0),

    CONSTRAINT products_price_nonnegative
        CHECK (price >= 0),

    CONSTRAINT products_status_valid
        CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX products_sku_lower_uidx
    ON products (LOWER(sku));


-- +goose Down
DROP TABLE products;
