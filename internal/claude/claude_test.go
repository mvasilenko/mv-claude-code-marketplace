package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform/mocks"
)

func TestNew_ClaudeNotFound(t *testing.T) {
	plat := mocks.NewMockPlatform(t)
	plat.EXPECT().CommandExists("claude").Return(false)

	cli, err := New("/test/.claude", plat)

	assert.Nil(t, cli)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNew_ClaudeFound(t *testing.T) {
	plat := mocks.NewMockPlatform(t)
	plat.EXPECT().CommandExists("claude").Return(true)

	cli, err := New("/test/.claude", plat)

	assert.NoError(t, err)
	assert.NotNil(t, cli)
}
