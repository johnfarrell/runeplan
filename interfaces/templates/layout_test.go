package templates_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/johnfarrell/runeplan/interfaces/templates"
)

func TestBase_ContainsTitle(t *testing.T) {
	var buf bytes.Buffer
	if err := templates.Base("My Page").Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "My Page") {
		t.Error("expected title in rendered HTML")
	}
	if !strings.Contains(buf.String(), "RunePlan") {
		t.Error("expected brand name in rendered HTML")
	}
}
