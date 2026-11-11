-- +goose Up
CREATE TABLE customers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT customers_name_not_empty
        CHECK (LENGTH(TRIM(name)) > 0)
);

CREATE UNIQUE INDEX customers_email_lower_uidx
    ON customers (LOWER(email));


-- +goose Down
DROP TABLE customers;
