CREATE TABLE customers (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  city TEXT NOT NULL
);

INSERT INTO customers VALUES (1, 'Alice', 'alice@example.com', 'Seattle');
INSERT INTO customers VALUES (2, 'Bob', 'bob@example.com', 'Portland');
INSERT INTO customers VALUES (3, 'Carol', 'carol@example.com', 'Seattle');
INSERT INTO customers VALUES (4, 'Dave', 'dave@example.com', 'Denver');
INSERT INTO customers VALUES (5, 'Eve', 'eve@example.com', 'Portland');

CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  customer_id INTEGER NOT NULL,
  product TEXT NOT NULL,
  amount INTEGER NOT NULL,
  order_date DATE NOT NULL
);

INSERT INTO orders VALUES (1, 1, 'Widget', 2, '2026-01-15');
INSERT INTO orders VALUES (2, 3, 'Gadget', 1, '2026-01-20');
INSERT INTO orders VALUES (3, 2, 'Widget', 5, '2026-02-01');
INSERT INTO orders VALUES (4, 1, 'Doohickey', 3, '2026-02-05');
INSERT INTO orders VALUES (5, 5, 'Gadget', 2, '2026-02-08');
INSERT INTO orders VALUES (6, 4, 'Widget', 1, '2026-02-10');
