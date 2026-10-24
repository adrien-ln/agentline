package render

import (
	"os"
	"testing"

	"github.com/adrien-ln/agentline/internal/i18n"
)

// TestMain : les références de tests/golden/ sont en français (celles de tests/golden/en/ passent par
// TestReferencesAnglaises).
func TestMain(m *testing.M) {
	i18n.Choisir(i18n.FR)
	os.Exit(m.Run())
}
