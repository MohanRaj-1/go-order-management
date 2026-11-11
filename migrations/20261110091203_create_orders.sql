-- +goose Up
CREATE TABLE orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT orders_customer_fk
        FOREIGN KEY (customer_id)
        REFERENCES customers(id)
        ON DELETE RESTRICT,

    CONSTRAINT orders_status_valid
        CHECK (status IN ('PENDING', 'CONFIRMED', 'CANCELLED', 'FULFILLED'))
);

CREATE INDEX orders_customer_id_idx
    ON orders (customer_id);


-- +goose Down
DROP TABLE orders;
