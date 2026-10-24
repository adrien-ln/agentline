package style

import (
	"os"
	"testing"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// TestMain : les textes attendus par les tests sont en français.
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	os.Exit(m.Run())
}
