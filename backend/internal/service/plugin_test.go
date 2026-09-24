package service

import "testing"

func TestBuildPlugin(t *testing.T) {
	enabled := true
	item, err := buildPlugin(2, UpsertPluginInput{
		Name:    "limit-min",
		Plugin:  "rate-limiting",
		Enabled: &enabled,
		Config: map[string]interface{}{
			"minute":   100,
			"limit_by": "ip",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.SpaceID != 2 || item.Plugin != "rate-limiting" || !item.Enabled {
		t.Fatalf("plugin = %+v", item)
	}
	if _, err := buildPlugin(1, UpsertPluginInput{Name: "bad name", Plugin: "cors"}); err == nil {
		t.Fatal("expected invalid name")
	}
	if _, err := buildPlugin(1, UpsertPluginInput{Name: "ok", Plugin: "unknown"}); err == nil {
		t.Fatal("expected unsupported plugin")
	}
	if _, err := buildPlugin(1, UpsertPluginInput{Name: "ok", Plugin: "acl", Config: map[string]interface{}{"allow": "a", "deny": "b"}}); err == nil {
		t.Fatal("expected acl conflict")
	}
}
