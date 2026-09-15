package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"ai-multi-platform-release/backend-go/models"
)

func integrationBoolPointer(value bool) *bool { return &value }

func mustPhotographyEncryptedSecret(t *testing.T, value string) string {
	t.Helper()
	ciphertext, err := encryptPhotographySecret(value)
	if err != nil {
		t.Fatalf("encryptPhotographySecret() error = %v", err)
	}
	return ciphertext
}

func mustPhotographyDecryptedSecret(t *testing.T, value string) string {
	t.Helper()
	plaintext, err := decryptPhotographySecret(value)
	if err != nil {
		t.Fatalf("decryptPhotographySecret() error = %v", err)
	}
	return plaintext
}

func newPhotographyIntegrationTestConfig(t *testing.T) models.PhotographyIntegrationConfig {
	t.Helper()
	t.Setenv("INTEGRATION_ENCRYPTION_KEY", "test-integration-key")
	return models.PhotographyIntegrationConfig{
		ID:                       photographyIntegrationConfigID,
		AMapWebKeyEncrypted:      mustPhotographyEncryptedSecret(t, "old-amap-web"),
		AMapSecurityKeyEncrypted: mustPhotographyEncryptedSecret(t, "old-amap-security"),
		GoogleMapsKeyEncrypted:   mustPhotographyEncryptedSecret(t, "old-google"),
		OpenMeteoKeyEncrypted:    mustPhotographyEncryptedSecret(t, "old-open-meteo"),
		QWeatherKeyEncrypted:     mustPhotographyEncryptedSecret(t, "old-qweather"),
		QWeatherCredentialType:   "api_key",
		NOAAKeyEncrypted:         mustPhotographyEncryptedSecret(t, "old-noaa"),
		TideKeyEncrypted:         mustPhotographyEncryptedSecret(t, "old-tide"),
		OpenMeteoHost:            "old-open-meteo-host",
		QWeatherHost:             "old-qweather-host",
		NOAAHost:                 "old-noaa-host",
		TideHost:                 "old-tide-host",
		OpenMeteoEnabled:         true,
		QWeatherEnabled:          false,
		NOAAEnabled:              true,
		TideEnabled:              true,
	}
}

func TestApplyPhotographyIntegrationConfigBySection(t *testing.T) {
	tests := []struct {
		name    string
		payload PhotographyIntegrationConfigPayload
		assert  func(t *testing.T, config models.PhotographyIntegrationConfig)
	}{
		{
			name: "map only changes map fields",
			payload: PhotographyIntegrationConfigPayload{
				Section:            photographyIntegrationSectionMap,
				AMapWebKey:         "new-amap-web",
				AMapSecurityKey:    "new-amap-security",
				GoogleMapsKey:      "new-google",
				ClearGoogleMapsKey: true,
				OpenMeteoHost:      "",
				QWeatherHost:       "",
				NOAAHost:           "",
				TideHost:           "",
			},
			assert: func(t *testing.T, config models.PhotographyIntegrationConfig) {
				if got := mustPhotographyDecryptedSecret(t, config.AMapWebKeyEncrypted); got != "new-amap-web" {
					t.Fatalf("AMapWebKey = %q", got)
				}
				if got := mustPhotographyDecryptedSecret(t, config.AMapSecurityKeyEncrypted); got != "new-amap-security" {
					t.Fatalf("AMapSecurityKey = %q", got)
				}
				if config.GoogleMapsKeyEncrypted != "" {
					t.Fatalf("GoogleMapsKey should be cleared, got %q", config.GoogleMapsKeyEncrypted)
				}
				if got := mustPhotographyDecryptedSecret(t, config.OpenMeteoKeyEncrypted); got != "old-open-meteo" || config.OpenMeteoHost != "old-open-meteo-host" {
					t.Fatalf("weather config changed: key=%q host=%q", got, config.OpenMeteoHost)
				}
				if got := mustPhotographyDecryptedSecret(t, config.TideKeyEncrypted); got != "old-tide" || config.TideHost != "old-tide-host" || !config.TideEnabled {
					t.Fatalf("tide config changed: key=%q host=%q enabled=%v", got, config.TideHost, config.TideEnabled)
				}
			},
		},
		{
			name: "weather only changes weather fields",
			payload: PhotographyIntegrationConfigPayload{
				Section:                photographyIntegrationSectionWeather,
				OpenMeteoKey:           "new-open-meteo",
				QWeatherKey:            "new-qweather",
				QWeatherCredentialType: "token",
				OpenMeteoHost:          "new-open-meteo-host",
				QWeatherHost:           "new-qweather-host",
				OpenMeteoEnabled:       integrationBoolPointer(false),
				QWeatherEnabled:        integrationBoolPointer(true),
			},
			assert: func(t *testing.T, config models.PhotographyIntegrationConfig) {
				if got := mustPhotographyDecryptedSecret(t, config.OpenMeteoKeyEncrypted); got != "new-open-meteo" || config.OpenMeteoHost != "new-open-meteo-host" || config.OpenMeteoEnabled {
					t.Fatalf("Open-Meteo config = key %q host %q enabled %v", got, config.OpenMeteoHost, config.OpenMeteoEnabled)
				}
				if got := mustPhotographyDecryptedSecret(t, config.QWeatherKeyEncrypted); got != "new-qweather" || config.QWeatherHost != "new-qweather-host" || config.QWeatherCredentialType != "token" || !config.QWeatherEnabled {
					t.Fatalf("QWeather config = key %q host %q credential %q enabled %v", got, config.QWeatherHost, config.QWeatherCredentialType, config.QWeatherEnabled)
				}
				if got := mustPhotographyDecryptedSecret(t, config.AMapWebKeyEncrypted); got != "old-amap-web" || config.TideHost != "old-tide-host" || !config.TideEnabled {
					t.Fatalf("non-weather config changed: map key=%q tide host=%q enabled=%v", got, config.TideHost, config.TideEnabled)
				}
			},
		},
		{
			name: "tide only changes tide fields",
			payload: PhotographyIntegrationConfigPayload{
				Section:     photographyIntegrationSectionTide,
				TideKey:     "new-tide",
				TideHost:    "new-tide-host",
				TideEnabled: integrationBoolPointer(false),
			},
			assert: func(t *testing.T, config models.PhotographyIntegrationConfig) {
				if got := mustPhotographyDecryptedSecret(t, config.TideKeyEncrypted); got != "new-tide" || config.TideHost != "new-tide-host" || config.TideEnabled {
					t.Fatalf("tide config = key %q host %q enabled %v", got, config.TideHost, config.TideEnabled)
				}
				if got := mustPhotographyDecryptedSecret(t, config.NOAAKeyEncrypted); got != "old-noaa" || config.NOAAHost != "old-noaa-host" || !config.NOAAEnabled {
					t.Fatalf("aurora config changed: key=%q host=%q enabled=%v", got, config.NOAAHost, config.NOAAEnabled)
				}
			},
		},
		{
			name: "aurora only changes aurora fields",
			payload: PhotographyIntegrationConfigPayload{
				Section:     photographyIntegrationSectionAurora,
				NOAAKey:     "new-noaa",
				NOAAHost:    "new-noaa-host",
				NOAAEnabled: integrationBoolPointer(false),
			},
			assert: func(t *testing.T, config models.PhotographyIntegrationConfig) {
				if got := mustPhotographyDecryptedSecret(t, config.NOAAKeyEncrypted); got != "new-noaa" || config.NOAAHost != "new-noaa-host" || config.NOAAEnabled {
					t.Fatalf("aurora config = key %q host %q enabled %v", got, config.NOAAHost, config.NOAAEnabled)
				}
				if got := mustPhotographyDecryptedSecret(t, config.TideKeyEncrypted); got != "old-tide" || config.TideHost != "old-tide-host" || !config.TideEnabled {
					t.Fatalf("non-aurora config changed: key=%q host=%q enabled=%v", got, config.TideHost, config.TideEnabled)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := newPhotographyIntegrationTestConfig(t)
			if err := applyPhotographyIntegrationConfig(&config, tt.payload); err != nil {
				t.Fatalf("applyPhotographyIntegrationConfig() error = %v", err)
			}
			tt.assert(t, config)
		})
	}
}

func TestIntegrationEncryptionKeyAutoGeneratesAndReusesLocalKey(t *testing.T) {
	t.Setenv("INTEGRATION_ENCRYPTION_KEY", "")
	keyFile := filepath.Join(t.TempDir(), "integration.key")
	t.Setenv("INTEGRATION_ENCRYPTION_KEY_FILE", keyFile)

	first, err := integrationEncryptionKey()
	if err != nil {
		t.Fatalf("integrationEncryptionKey() error = %v", err)
	}
	second, err := integrationEncryptionKey()
	if err != nil {
		t.Fatalf("integrationEncryptionKey() second call error = %v", err)
	}
	if len(first) != 32 || !bytes.Equal(first, second) {
		t.Fatalf("generated key was not stable 32-byte key: first=%d second=%d", len(first), len(second))
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatalf("generated key file stat error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("generated key file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyPhotographyIntegrationConfigKeepsFullPayloadCompatibility(t *testing.T) {
	config := newPhotographyIntegrationTestConfig(t)
	payload := PhotographyIntegrationConfigPayload{
		AMapWebKey:             "new-amap",
		AMapSecurityKey:        "new-security",
		GoogleMapsKey:          "new-google",
		OpenMeteoKey:           "new-open-meteo",
		QWeatherKey:            "new-qweather",
		QWeatherCredentialType: "token",
		NOAAKey:                "new-noaa",
		TideKey:                "new-tide",
		OpenMeteoHost:          "new-open-meteo-host",
		QWeatherHost:           "new-qweather-host",
		NOAAHost:               "new-noaa-host",
		TideHost:               "new-tide-host",
		OpenMeteoEnabled:       integrationBoolPointer(false),
		QWeatherEnabled:        integrationBoolPointer(true),
		NOAAEnabled:            integrationBoolPointer(false),
		TideEnabled:            integrationBoolPointer(false),
	}
	if err := applyPhotographyIntegrationConfig(&config, payload); err != nil {
		t.Fatalf("applyPhotographyIntegrationConfig() error = %v", err)
	}
	if got := mustPhotographyDecryptedSecret(t, config.AMapWebKeyEncrypted); got != "new-amap" || config.TideHost != "new-tide-host" || config.TideEnabled {
		t.Fatalf("full payload did not update all fields: key=%q tideHost=%q tideEnabled=%v", got, config.TideHost, config.TideEnabled)
	}
	if config.QWeatherCredentialType != "token" || !config.QWeatherEnabled || config.OpenMeteoEnabled || config.NOAAEnabled {
		t.Fatalf("full payload flags = %+v", config)
	}
}

func TestApplyPhotographyIntegrationConfigValidatesSectionAndHosts(t *testing.T) {
	config := newPhotographyIntegrationTestConfig(t)
	if err := applyPhotographyIntegrationConfig(&config, PhotographyIntegrationConfigPayload{Section: "unknown"}); err == nil || !IsPhotographyValidationError(err) {
		t.Fatalf("invalid section error = %v", err)
	}
	longHost := "https://" + string(make([]byte, 501))
	if err := applyPhotographyIntegrationConfig(&config, PhotographyIntegrationConfigPayload{
		Section:       photographyIntegrationSectionWeather,
		OpenMeteoHost: longHost,
	}); err == nil || !IsPhotographyValidationError(err) {
		t.Fatalf("long host error = %v", err)
	}
}

func TestPhotographyIntegrationConfigLoadAndSaveUsesDefaultPrimaryKey(t *testing.T) {
	t.Setenv("INTEGRATION_ENCRYPTION_KEY", "test-integration-key")
	if err := InitDatabase(filepath.Join(t.TempDir(), "photography-integration.db")); err != nil {
		t.Fatalf("InitDatabase() error = %v", err)
	}
	// 该测试会切换包级 ORM 到临时数据库，结束后恢复为空，避免污染同包天气服务测试。
	t.Cleanup(func() {
		defaultOrm = nil
		defaultDBPath = ""
	})

	loaded, err := LoadPhotographyIntegrationConfig()
	if err != nil {
		t.Fatalf("LoadPhotographyIntegrationConfig() on empty database error = %v", err)
	}
	if loaded.ID != "" {
		t.Fatalf("empty database config ID = %q, want empty ID for first insert", loaded.ID)
	}

	if err := SavePhotographyIntegrationConfig(PhotographyIntegrationConfigPayload{
		Section:    photographyIntegrationSectionMap,
		AMapWebKey: "first-amap-key",
	}, "test-user"); err != nil {
		t.Fatalf("first map save error = %v", err)
	}
	loaded, err = LoadPhotographyIntegrationConfig()
	if err != nil {
		t.Fatalf("LoadPhotographyIntegrationConfig() after first save error = %v", err)
	}
	if loaded.ID != photographyIntegrationConfigID {
		t.Fatalf("saved config ID = %q, want %q", loaded.ID, photographyIntegrationConfigID)
	}
	if got := mustPhotographyDecryptedSecret(t, loaded.AMapWebKeyEncrypted); got != "first-amap-key" {
		t.Fatalf("saved map key = %q", got)
	}

	count, err := GetOrm().QueryTable(new(models.PhotographyIntegrationConfig)).Count()
	if err != nil {
		t.Fatalf("count integration configs error = %v", err)
	}
	if count != 1 {
		t.Fatalf("integration config row count = %d, want 1", count)
	}

	if err := SavePhotographyIntegrationConfig(PhotographyIntegrationConfigPayload{
		Section:          photographyIntegrationSectionWeather,
		OpenMeteoHost:    "https://weather.example.test",
		OpenMeteoEnabled: integrationBoolPointer(false),
	}, "test-user"); err != nil {
		t.Fatalf("weather update error = %v", err)
	}
	loaded, err = LoadPhotographyIntegrationConfig()
	if err != nil {
		t.Fatalf("LoadPhotographyIntegrationConfig() after update error = %v", err)
	}
	if loaded.OpenMeteoHost != "https://weather.example.test" || loaded.OpenMeteoEnabled {
		t.Fatalf("weather update = host %q enabled %v", loaded.OpenMeteoHost, loaded.OpenMeteoEnabled)
	}
	if got := mustPhotographyDecryptedSecret(t, loaded.AMapWebKeyEncrypted); got != "first-amap-key" {
		t.Fatalf("map key changed after weather update = %q", got)
	}
	count, err = GetOrm().QueryTable(new(models.PhotographyIntegrationConfig)).Count()
	if err != nil {
		t.Fatalf("count integration configs after update error = %v", err)
	}
	if count != 1 {
		t.Fatalf("integration config row count after update = %d, want 1", count)
	}
}
