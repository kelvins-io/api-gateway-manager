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

func TestBuildPluginAIRouter(t *testing.T) {
	item, err := buildPlugin(1, UpsertPluginInput{
		Name:   "llm-router",
		Plugin: "ai-router",
		Config: map[string]interface{}{
			"model_list": []interface{}{
				map[string]interface{}{
					"model_name": "gpt-4o",
					"catch_all":  true,
					"litellm_params": map[string]interface{}{
						"model": "openai/gpt-4o",
					},
				},
			},
			"routing_strategy": "simple-shuffle",
			"state_backend":    "local",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Plugin != "ai-router" {
		t.Fatalf("plugin = %s", item.Plugin)
	}

	if _, err := buildPlugin(1, UpsertPluginInput{
		Name:   "llm-bad",
		Plugin: "ai-router",
		Config: map[string]interface{}{
			"routing_strategy": "simple-shuffle",
			"model_list":       []interface{}{},
		},
	}); err == nil {
		t.Fatal("expected empty model_list to fail")
	}

	if _, err := buildPlugin(1, UpsertPluginInput{
		Name:   "llm-redis",
		Plugin: "ai-router",
		Config: map[string]interface{}{
			"model_list": []interface{}{
				map[string]interface{}{"model_name": "a", "litellm_params": map[string]interface{}{"model": "openai/a"}},
			},
			"state_backend": "redis",
			"redis":         map[string]interface{}{},
		},
	}); err == nil {
		t.Fatal("expected redis.host required")
	}

	if _, err := buildPlugin(1, UpsertPluginInput{
		Name:   "llm-semantic",
		Plugin: "ai-router",
		Config: map[string]interface{}{
			"model_list": []interface{}{
				map[string]interface{}{"model_name": "a", "litellm_params": map[string]interface{}{"model": "openai/a"}},
			},
			"routing_strategy": "semantic",
			"semantic":         map[string]interface{}{},
		},
	}); err == nil {
		t.Fatal("expected semantic.api_key required")
	}
}
