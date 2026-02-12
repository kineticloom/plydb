CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    city TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    order_date DATE NOT NULL
);

INSERT INTO customers (id, name, email, city) VALUES
    (1, 'Alice', 'alice@example.com', 'Seattle'),
    (2, 'Bob', 'bob@example.com', 'Portland'),
    (3, 'Carol', 'carol@example.com', 'Seattle'),
    (4, 'Dave', 'dave@example.com', 'Denver'),
    (5, 'Eve', 'eve@example.com', 'Portland');

INSERT INTO orders (id, customer_id, product_id, quantity, order_date) VALUES
    (1, 1, 1, 2, '2026-01-15'),
    (2, 3, 2, 1, '2026-01-20'),
    (3, 2, 1, 5, '2026-02-01'),
    (4, 1, 4, 3, '2026-02-05'),
    (5, 5, 2, 2, '2026-02-08'),
    (6, 4, 3, 1, '2026-02-10'),
    (7, 3, 5, 10, '2026-02-11'),
    (8, 2, 4, 1, '2026-02-11');
