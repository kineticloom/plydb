//go:build integration

package queryengine

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ---------------------------------------------------------------------------
// Container helpers
// ---------------------------------------------------------------------------

func pgContainer(t *testing.T) (host string, port int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "postgres",
			func(host string, port nat.Port) string {
				return fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
			}).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	mappedHost, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting postgres host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("getting postgres port: %v", err)
	}

	return mappedHost, mappedPort.Int(), func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: terminating postgres container: %v", err)
		}
	}
}

func mysqlContainer(t *testing.T) (host string, port int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "testpass",
			"MYSQL_DATABASE":      "testdb",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "testpass",
		},
		WaitingFor: wait.ForSQL("3306/tcp", "mysql",
			func(host string, port nat.Port) string {
				return fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb", host, port.Port())
			}).WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting mysql container: %v", err)
	}

	mappedHost, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting mysql host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatalf("getting mysql port: %v", err)
	}

	return mappedHost, mappedPort.Int(), func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: terminating mysql container: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

func seedPostgres(t *testing.T, host string, port int) {
	t.Helper()

	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%d/testdb?sslmode=disable", host, port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("connecting to postgres for seeding: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS departments (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`INSERT INTO departments (id, name) VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,

		`CREATE TABLE IF NOT EXISTS employees (
			id            INTEGER PRIMARY KEY,
			name          TEXT NOT NULL,
			department_id INTEGER NOT NULL,
			salary        NUMERIC(10,2) NOT NULL
		)`,
		`INSERT INTO employees (id, name, department_id, salary) VALUES
			(1, 'Alice',   1, 95000.00),
			(2, 'Bob',     1, 88000.00),
			(3, 'Charlie', 2, 72000.00),
			(4, 'Diana',   3, 68000.00),
			(5, 'Eve',     2, 75000.00)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seeding postgres: %v\nstatement: %s", err, stmt)
		}
	}
}

func seedMySQL(t *testing.T, host string, port int) {
	t.Helper()

	dsn := fmt.Sprintf("testuser:testpass@tcp(%s:%d)/testdb", host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("connecting to mysql for seeding: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS departments (
			id   INT PRIMARY KEY,
			name VARCHAR(100) NOT NULL
		)`,
		`INSERT INTO departments (id, name) VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,

		`CREATE TABLE IF NOT EXISTS employees (
			id            INT PRIMARY KEY,
			name          VARCHAR(100) NOT NULL,
			department_id INT NOT NULL,
			salary        DECIMAL(10,2) NOT NULL
		)`,
		`INSERT INTO employees (id, name, department_id, salary) VALUES
			(1, 'Alice',   1, 95000.00),
			(2, 'Bob',     1, 88000.00),
			(3, 'Charlie', 2, 72000.00),
			(4, 'Diana',   3, 68000.00),
			(5, 'Eve',     2, 75000.00)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seeding mysql: %v\nstatement: %s", err, stmt)
		}
	}
}

// ---------------------------------------------------------------------------
// Config builders
// ---------------------------------------------------------------------------

const testPasswordEnvVar = "INTTEST_DB_PASS"

func pgConfig(t *testing.T, key, host string, port int) *Config {
	t.Helper()
	t.Setenv(testPasswordEnvVar, "testpass")
	return &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			key: {
				Metadata:       Metadata{Name: "Integration Postgres", Description: "test"},
				Type:           PostgreSQL,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
			},
		},
	}
}

func mysqlConfig(t *testing.T, key, host string, port int) *Config {
	t.Helper()
	t.Setenv(testPasswordEnvVar, "testpass")
	return &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			key: {
				Metadata:       Metadata{Name: "Integration MySQL", Description: "test"},
				Type:           MySQL,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
			},
		},
	}
}

func mixedConfig(t *testing.T, dbKey string, dbType DatabaseType, host string, port int) *Config {
	t.Helper()
	t.Setenv(testPasswordEnvVar, "testpass")

	csvPath := testdataPath("integration_products.csv")

	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			dbKey: {
				Metadata:       Metadata{Name: "Integration DB", Description: "test"},
				Type:           dbType,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
			},
			"products_csv": {
				Metadata: Metadata{Name: "Products CSV", Description: "test csv"},
				Type:     File,
				Path:     csvPath,
				Format:   "csv",
			},
		},
	}
	return cfg
}

// ---------------------------------------------------------------------------
// Postgres integration tests
// ---------------------------------------------------------------------------

func TestIntegrationPostgres(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	seedPostgres(t, host, port)

	t.Run("BasicSelect", func(t *testing.T) {
		cfg := pgConfig(t, "pg", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT id, name, department_id, salary FROM "pg".public.employees ORDER BY id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type employee struct {
			id     int
			name   string
			deptID int
			salary float64
		}

		want := []employee{
			{1, "Alice", 1, 95000.00},
			{2, "Bob", 1, 88000.00},
			{3, "Charlie", 2, 72000.00},
			{4, "Diana", 3, 68000.00},
			{5, "Eve", 2, 75000.00},
		}

		var got []employee
		for rows.Next() {
			var e employee
			if err := rows.Scan(&e.id, &e.name, &e.deptID, &e.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, e)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("Filter", func(t *testing.T) {
		cfg := pgConfig(t, "pg", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT name, salary FROM "pg".public.employees WHERE salary > 80000 ORDER BY salary DESC`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name   string
			salary float64
		}

		want := []result{
			{"Alice", 95000.00},
			{"Bob", 88000.00},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("Aggregation", func(t *testing.T) {
		cfg := pgConfig(t, "pg", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT department_id, COUNT(*) AS cnt, AVG(salary) AS avg_sal
			 FROM "pg".public.employees
			 GROUP BY department_id
			 ORDER BY department_id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			deptID int
			cnt    int
			avgSal float64
		}

		want := []result{
			{1, 2, 91500.00},
			{2, 2, 73500.00},
			{3, 1, 68000.00},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.deptID, &r.cnt, &r.avgSal); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("JoinWithinDB", func(t *testing.T) {
		cfg := pgConfig(t, "pg", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT e.name, d.name AS dept
			 FROM "pg".public.employees e
			 JOIN "pg".public.departments d ON e.department_id = d.id
			 ORDER BY e.id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name string
			dept string
		}

		want := []result{
			{"Alice", "Engineering"},
			{"Bob", "Engineering"},
			{"Charlie", "Sales"},
			{"Diana", "Marketing"},
			{"Eve", "Sales"},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.dept); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("PreprocessThenQuery", func(t *testing.T) {
		cfg := pgConfig(t, "pg", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		original := `SELECT name, salary FROM pg.public.employees WHERE salary >= 75000 ORDER BY name`
		rewritten, err := PreprocessQuery(original, cfg)
		if err != nil {
			t.Fatalf("PreprocessQuery: %v", err)
		}

		rows, err := engine.Query(context.Background(), rewritten)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			var salary float64
			if err := rows.Scan(&name, &salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		wantNames := []string{"Alice", "Bob", "Eve"}
		if len(names) != len(wantNames) {
			t.Fatalf("got %d rows, want %d", len(names), len(wantNames))
		}
		for i := range wantNames {
			if names[i] != wantNames[i] {
				t.Errorf("row %d: got %q, want %q", i, names[i], wantNames[i])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// MySQL integration tests
// ---------------------------------------------------------------------------

func TestIntegrationMySQL(t *testing.T) {
	host, port, cleanup := mysqlContainer(t)
	defer cleanup()

	seedMySQL(t, host, port)

	t.Run("BasicSelect", func(t *testing.T) {
		cfg := mysqlConfig(t, "my", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT id, name, department_id, salary FROM "my".testdb.employees ORDER BY id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type employee struct {
			id     int
			name   string
			deptID int
			salary float64
		}

		want := []employee{
			{1, "Alice", 1, 95000.00},
			{2, "Bob", 1, 88000.00},
			{3, "Charlie", 2, 72000.00},
			{4, "Diana", 3, 68000.00},
			{5, "Eve", 2, 75000.00},
		}

		var got []employee
		for rows.Next() {
			var e employee
			if err := rows.Scan(&e.id, &e.name, &e.deptID, &e.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, e)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("Filter", func(t *testing.T) {
		cfg := mysqlConfig(t, "my", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT name, salary FROM "my".testdb.employees WHERE salary > 80000 ORDER BY salary DESC`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name   string
			salary float64
		}

		want := []result{
			{"Alice", 95000.00},
			{"Bob", 88000.00},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("Aggregation", func(t *testing.T) {
		cfg := mysqlConfig(t, "my", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT department_id, COUNT(*) AS cnt, AVG(salary) AS avg_sal
			 FROM "my".testdb.employees
			 GROUP BY department_id
			 ORDER BY department_id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			deptID int
			cnt    int
			avgSal float64
		}

		want := []result{
			{1, 2, 91500.00},
			{2, 2, 73500.00},
			{3, 1, 68000.00},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.deptID, &r.cnt, &r.avgSal); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("JoinWithinDB", func(t *testing.T) {
		cfg := mysqlConfig(t, "my", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT e.name, d.name AS dept
			 FROM "my".testdb.employees e
			 JOIN "my".testdb.departments d ON e.department_id = d.id
			 ORDER BY e.id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name string
			dept string
		}

		want := []result{
			{"Alice", "Engineering"},
			{"Bob", "Engineering"},
			{"Charlie", "Sales"},
			{"Diana", "Marketing"},
			{"Eve", "Sales"},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.dept); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("PreprocessThenQuery", func(t *testing.T) {
		cfg := mysqlConfig(t, "my", host, port)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		original := `SELECT name, salary FROM my.testdb.employees WHERE salary >= 75000 ORDER BY name`
		rewritten, err := PreprocessQuery(original, cfg)
		if err != nil {
			t.Fatalf("PreprocessQuery: %v", err)
		}

		rows, err := engine.Query(context.Background(), rewritten)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			var salary float64
			if err := rows.Scan(&name, &salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		wantNames := []string{"Alice", "Bob", "Eve"}
		if len(names) != len(wantNames) {
			t.Fatalf("got %d rows, want %d", len(names), len(wantNames))
		}
		for i := range wantNames {
			if names[i] != wantNames[i] {
				t.Errorf("row %d: got %q, want %q", i, names[i], wantNames[i])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Cross-source tests (DB + CSV join)
// ---------------------------------------------------------------------------

func TestIntegrationPostgresCSVJoin(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	seedPostgres(t, host, port)

	cfg := mixedConfig(t, "pg", PostgreSQL, host, port)
	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer engine.Close()

	// Use PreprocessQuery to rewrite the CSV reference, Postgres refs pass through.
	original := `SELECT e.name AS employee, p.name AS product, p.unit_price
		FROM pg.public.employees e
		CROSS JOIN products_csv.main.products p
		WHERE e.department_id = 1 AND p.category = 'electronics'
		ORDER BY e.name, p.name`

	// The CSV file source gets rewritten to a read_csv-style reference.
	// We need to query with the rewritten SQL.
	csvPath := testdataPath("integration_products.csv")
	rewritten, err := PreprocessQuery(original, cfg)
	if err != nil {
		t.Fatalf("PreprocessQuery: %v", err)
	}

	// Verify the CSV path was rewritten into the query.
	if !strings.Contains(rewritten, csvPath) {
		t.Fatalf("expected rewritten query to contain CSV path %q, got: %s", csvPath, rewritten)
	}

	rows, err := engine.Query(context.Background(), rewritten)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	type result struct {
		employee  string
		product   string
		unitPrice float64
	}

	// 2 engineering employees x 3 electronics products = 6 rows
	var got []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.employee, &r.product, &r.unitPrice); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration error: %v", err)
	}

	if len(got) != 6 {
		t.Fatalf("got %d rows, want 6", len(got))
	}

	// Spot-check first and last rows.
	if got[0].employee != "Alice" || got[0].product != "Keyboard" {
		t.Errorf("first row: got %+v, want Alice/Keyboard", got[0])
	}
	if got[len(got)-1].employee != "Bob" || got[len(got)-1].product != "Monitor" {
		t.Errorf("last row: got %+v, want Bob/Monitor", got[len(got)-1])
	}
}

func TestIntegrationMySQLCSVJoin(t *testing.T) {
	host, port, cleanup := mysqlContainer(t)
	defer cleanup()

	seedMySQL(t, host, port)

	cfg := mixedConfig(t, "my", MySQL, host, port)
	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer engine.Close()

	original := `SELECT e.name AS employee, p.name AS product, p.unit_price
		FROM my.testdb.employees e
		CROSS JOIN products_csv.main.products p
		WHERE e.department_id = 1 AND p.category = 'electronics'
		ORDER BY e.name, p.name`

	csvPath := testdataPath("integration_products.csv")
	rewritten, err := PreprocessQuery(original, cfg)
	if err != nil {
		t.Fatalf("PreprocessQuery: %v", err)
	}

	if !strings.Contains(rewritten, csvPath) {
		t.Fatalf("expected rewritten query to contain CSV path %q, got: %s", csvPath, rewritten)
	}

	rows, err := engine.Query(context.Background(), rewritten)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	type result struct {
		employee  string
		product   string
		unitPrice float64
	}

	var got []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.employee, &r.product, &r.unitPrice); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		got = append(got, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration error: %v", err)
	}

	if len(got) != 6 {
		t.Fatalf("got %d rows, want 6", len(got))
	}

	if got[0].employee != "Alice" || got[0].product != "Keyboard" {
		t.Errorf("first row: got %+v, want Alice/Keyboard", got[0])
	}
}

// ---------------------------------------------------------------------------
// Error / edge-case tests
// ---------------------------------------------------------------------------

func TestIntegrationConnectionRefused(t *testing.T) {
	t.Setenv(testPasswordEnvVar, "testpass")

	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			"bad": {
				Metadata:       Metadata{Name: "Bad", Description: "unreachable"},
				Type:           PostgreSQL,
				Host:           "127.0.0.1",
				Port:           19999, // nothing listening here
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
	if !strings.Contains(err.Error(), "attaching") {
		t.Fatalf("expected attach error, got: %v", err)
	}
}

func TestIntegrationWrongPassword(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	t.Setenv(testPasswordEnvVar, "wrongpassword")

	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			"pg": {
				Metadata:       Metadata{Name: "Bad Auth", Description: "wrong password"},
				Type:           PostgreSQL,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestIntegrationQueryNonexistentTable(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	cfg := pgConfig(t, "pg", host, port)
	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer engine.Close()

	_, err = engine.Query(context.Background(),
		`SELECT * FROM "pg".public.nonexistent_table_xyz`)
	if err == nil {
		t.Fatal("expected error for nonexistent table, got nil")
	}
}
