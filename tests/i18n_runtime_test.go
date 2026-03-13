package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestI18N_RuntimeCriticalKeysHaveRussianOverrides(t *testing.T) {
	ru := mustLoadLang(t, filepath.Join("..", "gui", "static", "i18n", "ru.json"))

	required := []string{
		"upload_progress",
		"upload_waiting",
		"upload_cancelled",
		"upload_settings",
		"upload_concurrency",
		"upload_concurrency_hint",
		"storage_usage",
		"storage_usage_hint",
		"delete_all_music",
	}

	for _, key := range required {
		value, ok := ru[key]
		if !ok {
			t.Fatalf("missing required ru i18n key: %s", key)
		}
		if strings.TrimSpace(value) == "" {
			t.Fatalf("empty required ru i18n key: %s", key)
		}
		if looksUntranslatedForRU(value) {
			t.Fatalf("required ru i18n key looks untranslated: %s=%q", key, value)
		}
	}
}
