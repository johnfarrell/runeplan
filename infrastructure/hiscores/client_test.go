package hiscores_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/infrastructure/hiscores"
)

// OSRS CSV format: rank,level,xp per skill in canonical order
const sampleCSV = `1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
1,99,200000000
`

func TestFetch_ParsesSkillLevels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleCSV))
	}))
	defer srv.Close()

	client := hiscores.NewClient(srv.URL, 0)
	levels, err := client.Fetch("Zezima")
	if err != nil {
		t.Fatal(err)
	}
	xp, ok := levels[skill.Attack]
	if !ok {
		t.Fatal("expected attack XP")
	}
	if xp.Value() != 200000000 {
		t.Errorf("got %d, want 200000000", xp.Value())
	}
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := hiscores.NewClient(srv.URL, 0)
	_, err := client.Fetch("unknownplayer")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
