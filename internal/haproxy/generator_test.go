package haproxy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/hapctl/internal/models"
)

func TestGenerateBindConfig(t *testing.T) {
	generator := NewGenerator("")

	t.Run("must generate TCP bind config", func(t *testing.T) {
		bind := &models.Bind{
			Name:        "test-tcp",
			Description: "Test TCP bind",
			Type:        "tcp",
			IP:          "*",
			Port:        8080,
			Backend: models.Backend{
				Servers: []models.Server{
					{Name: "server1", Address: "127.0.0.1:8080"},
				},
			},
		}

		config, err := generator.GenerateBindConfig(bind)
		require.NoError(t, err)
		require.Contains(t, config, "listen test-tcp")
		require.Contains(t, config, "bind *:8080")
		require.Contains(t, config, "mode tcp")
		require.Contains(t, config, "server server1 127.0.0.1:8080")
	})

	t.Run("must generate HTTP bind config", func(t *testing.T) {
		bind := &models.Bind{
			Name:        "test-http",
			Description: "Test HTTP bind",
			Type:        "http",
			IP:          "10.0.0.1",
			Port:        80,
			Backend: models.Backend{
				Servers: []models.Server{
					{Name: "web1", Address: "127.0.0.1:3000"},
					{Name: "web2", Address: "127.0.0.1:3001"},
				},
			},
		}

		config, err := generator.GenerateBindConfig(bind)
		require.NoError(t, err)
		require.Contains(t, config, "frontend test-http")
		require.Contains(t, config, "bind 10.0.0.1:80")
		require.Contains(t, config, "mode http")
		require.Contains(t, config, "backend test-http_backend")
		require.Contains(t, config, "server web1 127.0.0.1:3000")
		require.Contains(t, config, "server web2 127.0.0.1:3001")
	})

	t.Run("must fail with unsupported bind type", func(t *testing.T) {
		bind := &models.Bind{
			Name: "test-invalid",
			Type: "invalid",
			Port: 8080,
		}

		_, err := generator.GenerateBindConfig(bind)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported bind type")
	})
}

func TestFormatBindAddress(t *testing.T) {
	generator := NewGenerator("")

	t.Run("must format wildcard address", func(t *testing.T) {
		bind := &models.Bind{IP: "*", Port: 8080}
		addr := generator.formatBindAddress(bind)
		require.Equal(t, "*:8080", addr)
	})

	t.Run("must format empty IP as wildcard", func(t *testing.T) {
		bind := &models.Bind{IP: "", Port: 8080}
		addr := generator.formatBindAddress(bind)
		require.Equal(t, "*:8080", addr)
	})

	t.Run("must format specific IP", func(t *testing.T) {
		bind := &models.Bind{IP: "10.0.0.1", Port: 8080}
		addr := generator.formatBindAddress(bind)
		require.Equal(t, "10.0.0.1:8080", addr)
	})
}
