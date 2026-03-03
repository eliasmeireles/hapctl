package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceTemplate(t *testing.T) {
	t.Run("must contain required systemd sections", func(t *testing.T) {
		assert.Contains(t, defaultServiceTemplate, "[Unit]")
		assert.Contains(t, defaultServiceTemplate, "[Service]")
		assert.Contains(t, defaultServiceTemplate, "[Install]")
	})

	t.Run("must reference hapctl agent command", func(t *testing.T) {
		assert.Contains(t, defaultServiceTemplate, "/usr/local/bin/hapctl agent")
	})

	t.Run("must have restart configuration", func(t *testing.T) {
		assert.Contains(t, defaultServiceTemplate, "Restart=on-failure")
		assert.Contains(t, defaultServiceTemplate, "RestartSec=5s")
	})

	t.Run("must have security settings", func(t *testing.T) {
		assert.Contains(t, defaultServiceTemplate, "NoNewPrivileges=true")
		assert.Contains(t, defaultServiceTemplate, "ProtectSystem=strict")
	})
}
