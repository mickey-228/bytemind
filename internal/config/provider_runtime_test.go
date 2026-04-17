package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeProviderRuntimeConfigFile(t *testing.T, workspace string, cfg map[string]any) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	projectConfigDir := filepath.Join(workspace, ".bytemind")
	if err := os.MkdirAll(projectConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectConfigDir, "config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConfigLoadPreservesExplicitProviderRuntime(t *testing.T) {
	workspace := t.TempDir()
	writeProviderRuntimeConfigFile(t, workspace, map[string]any{
		"provider": map[string]any{
			"type":     "openai-compatible",
			"base_url": "https://api.openai.com/v1",
			"model":    "gpt-5.4-mini",
			"api_key":  "test-key",
		},
		"provider_runtime": map[string]any{
			"default_provider": "openai",
			"default_model":    "gpt-5.4-mini",
			"allow_fallback":   true,
			"providers": map[string]any{
				"openai": map[string]any{
					"type":     "openai-compatible",
					"base_url": "https://api.openai.com/v1",
					"model":    "gpt-5.4-mini",
					"api_key":  "test-key",
				},
			},
		},
	})
	cfg, err := Load(workspace, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProviderRuntime.DefaultProvider != "openai" || cfg.ProviderRuntime.DefaultModel != "gpt-5.4-mini" || !cfg.ProviderRuntime.AllowFallback {
		t.Fatalf("unexpected provider runtime %#v", cfg.ProviderRuntime)
	}
	if len(cfg.ProviderRuntime.Providers) != 1 {
		t.Fatalf("unexpected provider runtime providers %#v", cfg.ProviderRuntime.Providers)
	}
}

func TestLegacyProviderRuntimeConfigNormalizesProviderIDs(t *testing.T) {
	tests := []struct {
		name      string
		typeValue string
		want      string
	}{
		{name: "openai compatible", typeValue: "openai-compatible", want: "openai"},
		{name: "openai alias", typeValue: "openai", want: "openai"},
		{name: "empty defaults openai", typeValue: "", want: "openai"},
		{name: "openai uppercase", typeValue: "OPENAI", want: "openai"},
		{name: "openai compatible padded", typeValue: " OpenAI-Compatible ", want: "openai"},
		{name: "anthropic uppercase", typeValue: "ANTHROPIC", want: "anthropic"},
		{name: "anthropic", typeValue: "anthropic", want: "anthropic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ProviderConfig{Type: tt.typeValue, Model: "test-model"}
			runtime := LegacyProviderRuntimeConfig(cfg)
			if runtime.DefaultProvider != tt.want {
				t.Fatalf("unexpected default provider %q", runtime.DefaultProvider)
			}
			if runtime.DefaultModel != "test-model" {
				t.Fatalf("unexpected default model %q", runtime.DefaultModel)
			}
			if len(runtime.Providers) != 1 || runtime.Providers[tt.want].Type != tt.want {
				t.Fatalf("unexpected providers %#v", runtime.Providers)
			}
		})
	}
}
