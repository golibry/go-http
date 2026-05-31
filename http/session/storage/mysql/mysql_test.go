//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type StorageIntegrationSuite struct {
	suite.Suite
	db        *sql.DB
	store     *Storage
	tableName string
	ctx       context.Context
	container testcontainers.Container
}

func TestStorageIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StorageIntegrationSuite))
}

func (s *StorageIntegrationSuite) SetupSuite() {
	var err error
	s.ctx = context.Background()

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
	s.store = New(s.db, s.tableName)
	s.Require().NoError(s.store.Init(s.ctx))
}

func (s *StorageIntegrationSuite) TearDownSuite() {
	if s.db != nil {
		_, _ = s.db.ExecContext(
			s.ctx,
			fmt.Sprintf("DROP TABLE IF EXISTS `%s`", s.tableName),
		)
		_ = s.db.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(s.ctx)
	}
}

func (s *StorageIntegrationSuite) TestItCanSetGetAndExists() {
	id := "sess_a"
	data := []byte("hello world")

	err := s.store.Set(s.ctx, id, data, 10*time.Second)
	s.Require().NoError(err)

	s.True(s.store.Exists(s.ctx, id))

	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(data, got)
}

func (s *StorageIntegrationSuite) TestItHonorsUpsert() {
	id := "sess_b"
	err := s.store.Set(s.ctx, id, []byte("v1"), 60*time.Second)
	s.Require().NoError(err)

	err = s.store.Set(s.ctx, id, []byte("v2"), 60*time.Second)
	s.Require().NoError(err)

	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Equal([]byte("v2"), got)
}

func (s *StorageIntegrationSuite) TestItCanDelete() {
	id := "sess_c"
	err := s.store.Set(s.ctx, id, []byte("to-delete"), 60*time.Second)
	s.Require().NoError(err)

	s.Require().NoError(s.store.Delete(s.ctx, id))

	s.False(s.store.Exists(s.ctx, id))
	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *StorageIntegrationSuite) TestItExpiresAndCleansUp() {
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
		"SELECT COUNT(*) FROM `"+s.tableName+"` WHERE id IN (?, ?)",
		id1,
		id2,
	)
	s.Require().NoError(row.Scan(&count))
	s.Equal(0, count)
}
