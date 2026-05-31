//go:build integration

package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type StorageIntegrationSuite struct {
	suite.Suite
	client    *goredis.Client
	store     *Storage
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
		Image:        "redis:7",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(45 * time.Second),
	}
	c, err := testcontainers.GenericContainer(
		s.ctx,
		testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true},
	)
	s.Require().NoError(err)
	s.container = c

	host, err := c.Host(s.ctx)
	s.Require().NoError(err)
	port, err := c.MappedPort(s.ctx, "6379/tcp")
	s.Require().NoError(err)

	s.client = goredis.NewClient(&goredis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	deadline := time.Now().Add(45 * time.Second)
	for {
		err = s.client.Ping(s.ctx).Err()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			s.Require().NoError(err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	s.store = New(s.client, "session:")
	s.Require().NoError(s.store.Init(s.ctx))
}

func (s *StorageIntegrationSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
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

func (s *StorageIntegrationSuite) TestItHonorsOverwrite() {
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

func (s *StorageIntegrationSuite) TestItExpiresWithRedisTTL() {
	id := "sess_d"
	s.Require().NoError(s.store.Set(s.ctx, id, []byte("short"), time.Second))

	time.Sleep(1500 * time.Millisecond)

	s.False(s.store.Exists(s.ctx, id))
	got, err := s.store.Get(s.ctx, id)
	s.Require().NoError(err)
	s.Nil(got)
}
