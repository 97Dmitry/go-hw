-- +goose Up
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    category TEXT NOT NULL,
    month DATE NOT NULL CHECK (EXTRACT(DAY FROM month) = 1),
    limit_amount NUMERIC(14,2) NOT NULL CHECK (limit_amount > 0),
    UNIQUE (category, month)
);

-- +goose Down
DROP TABLE budgets;
