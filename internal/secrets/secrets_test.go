package secrets

import "testing"

func TestParseItems(t *testing.T) {
	data := []byte("# comment\nGITHUB_TOKEN\tid-1\n\nCUSTOM\tid-2\tapi-key\nbad-line\n")
	items := parseItems(data)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Var != "GITHUB_TOKEN" || items[0].ID != "id-1" || items[0].Field != "" {
		t.Errorf("item 0 = %+v", items[0])
	}
	if items[1].Var != "CUSTOM" || items[1].ID != "id-2" || items[1].Field != "api-key" {
		t.Errorf("item 1 = %+v", items[1])
	}
}
