package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/suite"

	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type MySQLStorageIntegrationSuite struct {
	suite.Suite
	db        *sql.DB
	store     *MySQLStorage
	tableName string
	ctx       context.Context
	container testcontainers.Container
}

func TestMySQLStorageIntegrationSuite(t *testing.T) {
	suite.Run(t, new(MySQLStorageIntegrationSuite))
}

func (s *MySQLStorageIntegrationSuite) SetupSuite() {
	var err error
	s.ctx = context.Background()

	// Start an ephemeral MariaDB container (no persistent volumes)
	req := testcontainers.ContainerRequest{
		Image:        "mariadb:11",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MARIADB_ROOT_PASSWORD": "secret",
			"MARIADB_DATABASE":      "testdb",
		},
		WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(45 * time.Second),
	}
	c, err := testcontainers.GenericContainer(
		s.ctx,
		testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true},
	)
	s.Require().NoError(err)
	s.container = c

	host, err := c.Host(s.ctx)
	s.Require().NoError(err)
	port, err := c.MappedPort(s.ctx, "3306/tcp")
	s.Require().NoError(err)

	dsn := fmt.Sprintf(
		"root:secret@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		host,
		port.Port(),
		"testdb",
	)

	s.db, err = sql.Open("mysql", dsn)
	s.Require().NoError(err)

	// Wait for DB to be really ready by retrying Ping
	deadline := time.Now().Add(45 * time.Second)
	for {
		err = s.db.PingContext(s.ctx)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			s.Require().NoError(err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// unique table per run
	s.tableName = "sessions_it_" + time.Now().UTC().Format("20060102_150405") + "_" + randSuffix(6)

	s.store = NewMySQLStorage(s.db, s.tableName)
	s.Require().NoError(s.store.Init(s.ctx))
}

func (s *MySQLStorageIntegrationSuite) TearDownSuite() {
	if s.db != nil {
		// best-effort drop
		_, _ = s.db.ExecContext(s.ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", s.tableName))
		_ = s.db.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(s.ctx)
	}
}

func (s *MySQLStorageIntegrationSuite) TestItCanSetGetAndExists() {
	id := "sess_" + randSuffix(8)
	data := []byte("hello world")

	// Set with 10s TTL
	err := s.store.Set(s.ctx, id, data, 10*time.Second)
	s.Require().NoError(err)

	// Exists should be true
	s.True(s.store.Exists(s.ctx, id))

	// Get should return the same data
	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(data, got)
}

func (s *MySQLStorageIntegrationSuite) TestItHonorsUpsert() {
	id := "sess_" + randSuffix(8)
	err := s.store.Set(s.ctx, id, []byte("v1"), 60*time.Second)
	s.Require().NoError(err)

	// Update same id with new data and TTL
	err = s.store.Set(s.ctx, id, []byte("v2"), 60*time.Second)
	s.Require().NoError(err)

	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal([]byte("v2"), got)
}

func (s *MySQLStorageIntegrationSuite) TestItCanDelete() {
	id := "sess_" + randSuffix(8)
	err := s.store.Set(s.ctx, id, []byte("to-delete"), 60*time.Second)
	s.Require().NoError(err)

	// Delete
	s.Require().NoError(s.store.Delete(s.ctx, id))

	// Now it should not exist
	s.False(s.store.Exists(s.ctx, id))
	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *MySQLStorageIntegrationSuite) TestItExpiresAndCleansUp() {
	id1 := "sess_" + randSuffix(8)
	id2 := "sess_" + randSuffix(8)

	// Short TTLs
	s.Require().NoError(s.store.Set(s.ctx, id1, []byte("short"), 1*time.Second))
	s.Require().NoError(s.store.Set(s.ctx, id2, []byte("short2"), 1*time.Second))

	// Wait to expire
	time.Sleep(1500 * time.Millisecond)

	// They should be considered non-existent (expired)
	s.False(s.store.Exists(s.ctx, id1))
	s.False(s.store.Exists(s.ctx, id2))

	// Run cleanup to remove records physically
	s.Require().NoError(s.store.Cleanup(s.ctx))

	// Validate table has no rows with those ids
	var count int
	row := s.db.QueryRowContext(
		s.ctx,
		"SELECT COUNT(*) FROM "+s.tableName+" WHERE id IN (?, ?)",
		id1,
		id2,
	)
	s.Require().NoError(row.Scan(&count))
	s.Equal(0, count)
}

func randSuffix(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback to time
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().Format("150405.000")))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
