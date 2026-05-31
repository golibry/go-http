//go:build integration

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgreSQLStorageIntegrationSuite struct {
	suite.Suite
	db        *sql.DB
	store     *PostgreSQLStorage
	tableName string
	ctx       context.Context
	container testcontainers.Container
}

func TestPostgreSQLStorageIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PostgreSQLStorageIntegrationSuite))
}

func (s *PostgreSQLStorageIntegrationSuite) SetupSuite() {
	var err error
	s.ctx = context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(45 * time.Second),
	}
	c, err := testcontainers.GenericContainer(
		s.ctx,
		testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true},
	)
	s.Require().NoError(err)
	s.container = c

	host, err := c.Host(s.ctx)
	s.Require().NoError(err)
	port, err := c.MappedPort(s.ctx, "5432/tcp")
	s.Require().NoError(err)

	dsn := fmt.Sprintf(
		"postgres://postgres:secret@%s:%s/testdb?sslmode=disable",
		host,
		port.Port(),
	)

	s.db, err = sql.Open("postgres", dsn)
	s.Require().NoError(err)

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

	s.tableName = "sessions_it"
	s.store = NewPostgreSQLStorage(s.db, s.tableName)
	s.Require().NoError(s.store.Init(s.ctx))
}

func (s *PostgreSQLStorageIntegrationSuite) TearDownSuite() {
	if s.db != nil {
		_, _ = s.db.ExecContext(
			s.ctx,
			fmt.Sprintf("DROP TABLE IF EXISTS %s", quotePostgreSQLTableName(s.tableName)),
		)
		_ = s.db.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(s.ctx)
	}
}

func (s *PostgreSQLStorageIntegrationSuite) TestItCanSetGetAndExists() {
	id := "sess_a"
	data := []byte("hello world")

	err := s.store.Set(s.ctx, id, data, 10*time.Second)
	s.Require().NoError(err)

	s.True(s.store.Exists(s.ctx, id))

	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(data, got)
}

func (s *PostgreSQLStorageIntegrationSuite) TestItHonorsUpsert() {
	id := "sess_b"
	err := s.store.Set(s.ctx, id, []byte("v1"), 60*time.Second)
	s.Require().NoError(err)

	err = s.store.Set(s.ctx, id, []byte("v2"), 60*time.Second)
	s.Require().NoError(err)

	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal([]byte("v2"), got)
}

func (s *PostgreSQLStorageIntegrationSuite) TestItCanDelete() {
	id := "sess_c"
	err := s.store.Set(s.ctx, id, []byte("to-delete"), 60*time.Second)
	s.Require().NoError(err)

	s.Require().NoError(s.store.Delete(s.ctx, id))

	s.False(s.store.Exists(s.ctx, id))
	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *PostgreSQLStorageIntegrationSuite) TestItExpiresAndCleansUp() {
	id1 := "sess_d1"
	id2 := "sess_d2"

	s.Require().NoError(s.store.Set(s.ctx, id1, []byte("short"), 1*time.Second))
	s.Require().NoError(s.store.Set(s.ctx, id2, []byte("short2"), 1*time.Second))

	time.Sleep(1500 * time.Millisecond)

	s.False(s.store.Exists(s.ctx, id1))
	s.False(s.store.Exists(s.ctx, id2))

	s.Require().NoError(s.store.Cleanup(s.ctx))

	var count int
	row := s.db.QueryRowContext(
		s.ctx,
		"SELECT COUNT(*) FROM "+quotePostgreSQLTableName(s.tableName)+" WHERE id IN ($1, $2)",
		id1,
		id2,
	)
	s.Require().NoError(row.Scan(&count))
	s.Equal(0, count)
}
