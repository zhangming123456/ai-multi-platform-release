package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ai-multi-platform-release/backend-go/models"
	"github.com/beego/beego/v2/client/orm"
)

const photographyIntegrationConfigID = "default"

const (
	photographyIntegrationSectionMap     = "map"
	photographyIntegrationSectionWeather = "weather"
	photographyIntegrationSectionTide    = "tide"
	photographyIntegrationSectionAurora  = "aurora"
)

type PhotographyIntegrationProviderView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	APIKeyMasked string `json:"api_key_masked"`
	APIHost      string `json:"api_host"`
	Configured   bool   `json:"configured"`
	Enabled      bool   `json:"enabled"`
	RequiresKey  bool   `json:"requires_key"`
}

type PhotographyIntegrationConfigView struct {
	AMapWebKeyMasked       string                               `json:"amap_web_key_masked"`
	AMapSecurityKeyMasked  string                               `json:"amap_security_key_masked"`
	GoogleMapsKeyMasked    string                               `json:"google_maps_key_masked"`
	OpenMeteoKeyMasked     string                               `json:"open_meteo_key_masked"`
	QWeatherKeyMasked      string                               `json:"qweather_key_masked"`
	QWeatherCredentialType string                               `json:"qweather_credential_type"`
	NOAAKeyMasked          string                               `json:"noaa_key_masked"`
	TideKeyMasked          string                               `json:"tide_key_masked"`
	OpenMeteoHost          string                               `json:"open_meteo_host"`
	QWeatherHost           string                               `json:"qweather_host"`
	NOAAHost               string                               `json:"noaa_host"`
	TideHost               string                               `json:"tide_host"`
	OpenMeteoEnabled       bool                                 `json:"open_meteo_enabled"`
	QWeatherEnabled        bool                                 `json:"qweather_enabled"`
	NOAAEnabled            bool                                 `json:"noaa_enabled"`
	TideEnabled            bool                                 `json:"tide_enabled"`
	WeatherProviders       []PhotographyIntegrationProviderView `json:"weather_providers"`
	TideProviders          []PhotographyIntegrationProviderView `json:"tide_providers"`
	AuroraProviders        []PhotographyIntegrationProviderView `json:"aurora_providers"`
}

type PhotographyIntegrationConfigPayload struct {
	AMapWebKey             string `json:"amap_web_key"`
	AMapSecurityKey        string `json:"amap_security_key"`
	GoogleMapsKey          string `json:"google_maps_key"`
	OpenMeteoKey           string `json:"open_meteo_key"`
	QWeatherKey            string `json:"qweather_key"`
	QWeatherCredentialType string `json:"qweather_credential_type"`
	NOAAKey                string `json:"noaa_key"`
	TideKey                string `json:"tide_key"`
	OpenMeteoHost          string `json:"open_meteo_host"`
	QWeatherHost           string `json:"qweather_host"`
	NOAAHost               string `json:"noaa_host"`
	TideHost               string `json:"tide_host"`
	Section                string `json:"section,omitempty"`
	OpenMeteoEnabled       *bool  `json:"open_meteo_enabled"`
	QWeatherEnabled        *bool  `json:"qweather_enabled"`
	NOAAEnabled            *bool  `json:"noaa_enabled"`
	TideEnabled            *bool  `json:"tide_enabled"`
	ClearAMapWebKey        bool   `json:"clear_amap_web_key"`
	ClearAMapSecurityKey   bool   `json:"clear_amap_security_key"`
	ClearGoogleMapsKey     bool   `json:"clear_google_maps_key"`
	ClearOpenMeteoKey      bool   `json:"clear_open_meteo_key"`
	ClearQWeatherKey       bool   `json:"clear_qweather_key"`
	ClearNOAAKey           bool   `json:"clear_noaa_key"`
	ClearTideKey           bool   `json:"clear_tide_key"`
}

type photographyIntegrationSecrets struct {
	AMapWebKey, AMapSecurityKey, GoogleMapsKey                  string
	OpenMeteoKey, QWeatherKey, NOAAKey, TideKey                 string
	QWeatherCredentialType                                      string
	OpenMeteoHost, QWeatherHost, NOAAHost, TideHost             string
	OpenMeteoEnabled, QWeatherEnabled, NOAAEnabled, TideEnabled bool
}

var integrationEncryptionKeyMu sync.Mutex

// integrationEncryptionKey 返回第三方密钥的 AES-256 加密主密钥。
// 生产环境优先使用 INTEGRATION_ENCRYPTION_KEY；本地开发未配置时，
// 会在数据库旁生成并复用一个权限为 0600 的随机密钥，避免首次保存高德/天气 Key 直接失败。
func integrationEncryptionKey() ([]byte, error) {
	value := strings.TrimSpace(os.Getenv("INTEGRATION_ENCRYPTION_KEY"))
	if value != "" {
		return normalizeIntegrationEncryptionKey(value)
	}

	integrationEncryptionKeyMu.Lock()
	defer integrationEncryptionKeyMu.Unlock()

	keyPath := strings.TrimSpace(os.Getenv("INTEGRATION_ENCRYPTION_KEY_FILE"))
	if keyPath == "" {
		if defaultDBPath != "" {
			keyPath = defaultDBPath + ".integration.key"
		} else {
			keyPath = filepath.Join("backend-go", ".integration_encryption_key")
		}
	}
	if data, err := os.ReadFile(keyPath); err == nil {
		key, normalizeErr := normalizeIntegrationEncryptionKey(strings.TrimSpace(string(data)))
		if normalizeErr != nil {
			return nil, fmt.Errorf("本地第三方密钥加密文件格式无效: %w", normalizeErr)
		}
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("读取第三方密钥加密文件失败: %w", err)
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("生成第三方密钥加密主密钥失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, fmt.Errorf("创建第三方密钥加密目录失败: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key)+"\n"), 0o600); err != nil {
		return nil, fmt.Errorf("保存第三方密钥加密主密钥失败: %w", err)
	}
	return key, nil
}

func normalizeIntegrationEncryptionKey(value string) ([]byte, error) {
	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if len([]byte(value)) == 32 {
		return []byte(value), nil
	}
	if value == "" {
		return nil, errors.New("加密主密钥不能为空")
	}
	// 允许部署环境使用口令式环境变量，同时固定导出 32 字节 AES-256 密钥。
	hash := sha256.Sum256([]byte(value))
	return hash[:], nil
}

func encryptPhotographySecret(value string) (string, error) {
	key, err := integrationEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptPhotographySecret(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	key, err := integrationEncryptionKey()
	if err != nil {
		return "", err
	}
	sealed, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", errors.New("摄影工具密钥密文格式无效")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("摄影工具密钥密文长度无效")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("摄影工具密钥解密失败")
	}
	return string(plaintext), nil
}

func LoadPhotographyIntegrationConfig() (*models.PhotographyIntegrationConfig, error) {
	if GetOrm() == nil {
		return nil, errors.New("数据库未初始化")
	}
	config := &models.PhotographyIntegrationConfig{ID: photographyIntegrationConfigID}
	if err := GetOrm().Read(config); err != nil {
		if errors.Is(err, orm.ErrNoRows) {
			// 返回空 ID，让保存逻辑识别为首次创建；查询对象本身已带固定主键，避免 ORM 报 missed pk value。
			return &models.PhotographyIntegrationConfig{}, nil
		}
		return nil, err
	}
	return config, nil
}

func photographyStoredSecret(encrypted string) string {
	if encrypted == "" {
		return ""
	}
	value, err := decryptPhotographySecret(encrypted)
	if err != nil {
		LogBackendError("photography-integration", "decrypt", "photography_integration_configs", 500, err.Error())
		return ""
	}
	return value
}

func photographyIntegrationSecretsForUse() photographyIntegrationSecrets {
	secrets := photographyIntegrationSecrets{
		AMapWebKey:             strings.TrimSpace(os.Getenv("AMAP_WEB_KEY")),
		AMapSecurityKey:        strings.TrimSpace(os.Getenv("AMAP_SECURITY_KEY")),
		GoogleMapsKey:          strings.TrimSpace(os.Getenv("GOOGLE_MAPS_BROWSER_KEY")),
		OpenMeteoKey:           strings.TrimSpace(os.Getenv("OPEN_METEO_API_KEY")),
		QWeatherKey:            strings.TrimSpace(os.Getenv("QWEATHER_API_KEY")),
		QWeatherCredentialType: "api_key",
		NOAAKey:                strings.TrimSpace(os.Getenv("NOAA_API_KEY")),
		TideKey:                strings.TrimSpace(os.Getenv("TIDE_API_KEY")),
		OpenMeteoHost:          strings.TrimSpace(os.Getenv("OPEN_METEO_FORECAST_BASE_URL")),
		QWeatherHost:           strings.TrimSpace(os.Getenv("QWEATHER_API_HOST")),
		NOAAHost:               strings.TrimSpace(os.Getenv("NOAA_SWPC_BASE_URL")),
		TideHost:               strings.TrimSpace(os.Getenv("TIDE_API_HOST")),
		OpenMeteoEnabled:       true, QWeatherEnabled: true, NOAAEnabled: true, TideEnabled: true,
	}
	if token := strings.TrimSpace(os.Getenv("QWEATHER_API_TOKEN")); token != "" {
		secrets.QWeatherKey = token
		secrets.QWeatherCredentialType = "token"
	}
	if config, err := LoadPhotographyIntegrationConfig(); err == nil && config != nil {
		if value := photographyStoredSecret(config.AMapWebKeyEncrypted); value != "" {
			secrets.AMapWebKey = value
		}
		if value := photographyStoredSecret(config.AMapSecurityKeyEncrypted); value != "" {
			secrets.AMapSecurityKey = value
		}
		if value := photographyStoredSecret(config.GoogleMapsKeyEncrypted); value != "" {
			secrets.GoogleMapsKey = value
		}
		if value := photographyStoredSecret(config.OpenMeteoKeyEncrypted); value != "" {
			secrets.OpenMeteoKey = value
		}
		if value := photographyStoredSecret(config.QWeatherKeyEncrypted); value != "" {
			secrets.QWeatherKey = value
		}
		if config.QWeatherCredentialType == "token" || config.QWeatherCredentialType == "api_key" {
			secrets.QWeatherCredentialType = config.QWeatherCredentialType
		}
		if value := photographyStoredSecret(config.NOAAKeyEncrypted); value != "" {
			secrets.NOAAKey = value
		}
		if value := photographyStoredSecret(config.TideKeyEncrypted); value != "" {
			secrets.TideKey = value
		}
		if config.OpenMeteoHost != "" {
			secrets.OpenMeteoHost = config.OpenMeteoHost
		}
		if config.QWeatherHost != "" {
			secrets.QWeatherHost = config.QWeatherHost
		}
		if config.NOAAHost != "" {
			secrets.NOAAHost = config.NOAAHost
		}
		if config.TideHost != "" {
			secrets.TideHost = config.TideHost
		}
		secrets.OpenMeteoEnabled = config.OpenMeteoEnabled
		secrets.QWeatherEnabled = config.QWeatherEnabled
		secrets.NOAAEnabled = config.NOAAEnabled
		secrets.TideEnabled = config.TideEnabled
	}
	return secrets
}

func integrationSecretForWeatherSource(source string) (string, string, bool) {
	secrets := photographyIntegrationSecretsForUse()
	switch source {
	case PhotographyWeatherSourceQWeather:
		return secrets.QWeatherKey, secrets.QWeatherHost, secrets.QWeatherEnabled
	default:
		return secrets.OpenMeteoKey, secrets.OpenMeteoHost, secrets.OpenMeteoEnabled
	}
}

func ListPhotographyIntegrationConfig() PhotographyIntegrationConfigView {
	secrets := photographyIntegrationSecretsForUse()
	return PhotographyIntegrationConfigView{
		AMapWebKeyMasked: maskPhotographyAPIKey(secrets.AMapWebKey), AMapSecurityKeyMasked: maskPhotographyAPIKey(secrets.AMapSecurityKey),
		GoogleMapsKeyMasked: maskPhotographyAPIKey(secrets.GoogleMapsKey), OpenMeteoKeyMasked: maskPhotographyAPIKey(secrets.OpenMeteoKey),
		QWeatherKeyMasked: maskPhotographyAPIKey(secrets.QWeatherKey), QWeatherCredentialType: secrets.QWeatherCredentialType, NOAAKeyMasked: maskPhotographyAPIKey(secrets.NOAAKey), TideKeyMasked: maskPhotographyAPIKey(secrets.TideKey),
		OpenMeteoHost: secrets.OpenMeteoHost, QWeatherHost: secrets.QWeatherHost, NOAAHost: secrets.NOAAHost, TideHost: secrets.TideHost,
		OpenMeteoEnabled: secrets.OpenMeteoEnabled, QWeatherEnabled: secrets.QWeatherEnabled, NOAAEnabled: secrets.NOAAEnabled, TideEnabled: secrets.TideEnabled,
		WeatherProviders: []PhotographyIntegrationProviderView{
			{ID: "open-meteo", Name: "Open-Meteo", APIKeyMasked: maskPhotographyAPIKey(secrets.OpenMeteoKey), APIHost: secrets.OpenMeteoHost, Configured: secrets.OpenMeteoKey != "" || secrets.OpenMeteoEnabled, Enabled: secrets.OpenMeteoEnabled, RequiresKey: false},
			{ID: "qweather", Name: "和风天气", APIKeyMasked: maskPhotographyAPIKey(secrets.QWeatherKey), APIHost: secrets.QWeatherHost, Configured: secrets.QWeatherKey != "" && secrets.QWeatherEnabled, Enabled: secrets.QWeatherEnabled, RequiresKey: true},
		},
		TideProviders:   []PhotographyIntegrationProviderView{{ID: "noaa-coops", Name: "NOAA CO-OPS", APIKeyMasked: maskPhotographyAPIKey(secrets.TideKey), APIHost: secrets.TideHost, Configured: secrets.TideEnabled, Enabled: secrets.TideEnabled, RequiresKey: false}, {ID: "stormglass", Name: "Stormglass", APIKeyMasked: maskPhotographyAPIKey(secrets.TideKey), APIHost: secrets.TideHost, Configured: secrets.TideKey != "" && secrets.TideEnabled, Enabled: secrets.TideEnabled, RequiresKey: true}},
		AuroraProviders: []PhotographyIntegrationProviderView{{ID: "noaa-swpc", Name: "NOAA SWPC", APIKeyMasked: maskPhotographyAPIKey(secrets.NOAAKey), APIHost: secrets.NOAAHost, Configured: secrets.NOAAEnabled, Enabled: secrets.NOAAEnabled, RequiresKey: false}},
	}
}

func updateEncryptedSecret(current *string, value string, clear bool) error {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "****") {
		value = ""
	}
	if clear {
		*current = ""
		return nil
	}
	if value == "" {
		return nil
	}
	encrypted, err := encryptPhotographySecret(value)
	if err != nil {
		return err
	}
	*current = encrypted
	return nil
}

func validatePhotographyIntegrationSection(section string) error {
	switch strings.TrimSpace(section) {
	case "", photographyIntegrationSectionMap, photographyIntegrationSectionWeather, photographyIntegrationSectionTide, photographyIntegrationSectionAurora:
		return nil
	default:
		return newPhotographyValidationError("摄影工具集成配置分区不合法")
	}
}

func validatePhotographyIntegrationHosts(payload PhotographyIntegrationConfigPayload) error {
	for _, host := range []string{payload.OpenMeteoHost, payload.QWeatherHost, payload.NOAAHost, payload.TideHost} {
		if len([]rune(strings.TrimSpace(host))) > 500 {
			return newPhotographyValidationError("API 地址不能超过 500 个字符")
		}
	}
	return nil
}

func applyPhotographyIntegrationCredentialType(config *models.PhotographyIntegrationConfig, payload PhotographyIntegrationConfigPayload) error {
	credentialType := strings.TrimSpace(payload.QWeatherCredentialType)
	if credentialType == "" {
		credentialType = config.QWeatherCredentialType
	}
	if credentialType == "" {
		credentialType = "api_key"
	}
	if credentialType != "api_key" && credentialType != "token" {
		return newPhotographyValidationError("和风天气认证方式必须为 api_key 或 token")
	}
	config.QWeatherCredentialType = credentialType
	return nil
}

func applyPhotographyIntegrationEnabled(config *models.PhotographyIntegrationConfig, payload PhotographyIntegrationConfigPayload) {
	if payload.OpenMeteoEnabled != nil {
		config.OpenMeteoEnabled = *payload.OpenMeteoEnabled
	}
	if payload.QWeatherEnabled != nil {
		config.QWeatherEnabled = *payload.QWeatherEnabled
	}
	if payload.NOAAEnabled != nil {
		config.NOAAEnabled = *payload.NOAAEnabled
	}
	if payload.TideEnabled != nil {
		config.TideEnabled = *payload.TideEnabled
	}
}

func applyPhotographyIntegrationConfig(config *models.PhotographyIntegrationConfig, payload PhotographyIntegrationConfigPayload) error {
	section := strings.TrimSpace(payload.Section)
	if err := validatePhotographyIntegrationSection(section); err != nil {
		return err
	}
	if err := validatePhotographyIntegrationHosts(payload); err != nil {
		return err
	}

	applyMap := section == "" || section == photographyIntegrationSectionMap
	applyWeather := section == "" || section == photographyIntegrationSectionWeather
	applyTide := section == "" || section == photographyIntegrationSectionTide
	applyAurora := section == "" || section == photographyIntegrationSectionAurora

	if applyMap {
		if err := updateEncryptedSecret(&config.AMapWebKeyEncrypted, payload.AMapWebKey, payload.ClearAMapWebKey); err != nil {
			return err
		}
		if err := updateEncryptedSecret(&config.AMapSecurityKeyEncrypted, payload.AMapSecurityKey, payload.ClearAMapSecurityKey); err != nil {
			return err
		}
		if err := updateEncryptedSecret(&config.GoogleMapsKeyEncrypted, payload.GoogleMapsKey, payload.ClearGoogleMapsKey); err != nil {
			return err
		}
	}

	if applyWeather {
		if err := updateEncryptedSecret(&config.OpenMeteoKeyEncrypted, payload.OpenMeteoKey, payload.ClearOpenMeteoKey); err != nil {
			return err
		}
		if err := updateEncryptedSecret(&config.QWeatherKeyEncrypted, payload.QWeatherKey, payload.ClearQWeatherKey); err != nil {
			return err
		}
		if err := applyPhotographyIntegrationCredentialType(config, payload); err != nil {
			return err
		}
		config.OpenMeteoHost = strings.TrimSpace(payload.OpenMeteoHost)
		config.QWeatherHost = strings.TrimSpace(payload.QWeatherHost)
	}

	if applyTide {
		if err := updateEncryptedSecret(&config.TideKeyEncrypted, payload.TideKey, payload.ClearTideKey); err != nil {
			return err
		}
		config.TideHost = strings.TrimSpace(payload.TideHost)
	}

	if applyAurora {
		if err := updateEncryptedSecret(&config.NOAAKeyEncrypted, payload.NOAAKey, payload.ClearNOAAKey); err != nil {
			return err
		}
		config.NOAAHost = strings.TrimSpace(payload.NOAAHost)
	}

	if section == "" {
		applyPhotographyIntegrationEnabled(config, payload)
	} else {
		switch section {
		case photographyIntegrationSectionWeather:
			applyPhotographyIntegrationEnabled(config, PhotographyIntegrationConfigPayload{
				OpenMeteoEnabled: payload.OpenMeteoEnabled,
				QWeatherEnabled:  payload.QWeatherEnabled,
			})
		case photographyIntegrationSectionTide:
			applyPhotographyIntegrationEnabled(config, PhotographyIntegrationConfigPayload{TideEnabled: payload.TideEnabled})
		case photographyIntegrationSectionAurora:
			applyPhotographyIntegrationEnabled(config, PhotographyIntegrationConfigPayload{NOAAEnabled: payload.NOAAEnabled})
		}
	}
	return nil
}

func SavePhotographyIntegrationConfig(payload PhotographyIntegrationConfigPayload, updatedBy string) error {
	if err := validatePhotographyIntegrationSection(payload.Section); err != nil {
		return err
	}
	if GetOrm() == nil {
		return errors.New("数据库未初始化")
	}
	config, err := LoadPhotographyIntegrationConfig()
	if err != nil {
		return err
	}
	newRecord := strings.TrimSpace(config.ID) == ""
	config.ID = photographyIntegrationConfigID
	if newRecord {
		config.OpenMeteoEnabled, config.NOAAEnabled, config.TideEnabled = true, true, true
	}
	if err := applyPhotographyIntegrationConfig(config, payload); err != nil {
		return err
	}
	config.UpdatedBy = strings.TrimSpace(updatedBy)
	if newRecord {
		_, err = GetOrm().Insert(config)
	} else {
		_, err = GetOrm().Update(config)
	}
	return err
}

type PhotographyMapConfig struct {
	Provider    string `json:"provider"`
	Configured  bool   `json:"configured"`
	BrowserKey  string `json:"browser_key"`
	SecurityKey string `json:"security_key"`
	Message     string `json:"message"`
}

func SelectPhotographyMapProvider(countryCode string) string {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if countryCode == "" || countryCode == "CN" {
		return "amap"
	}
	return "google"
}

func GetPhotographyMapConfig(countryCode string) PhotographyMapConfig {
	secrets := photographyIntegrationSecretsForUse()
	provider := SelectPhotographyMapProvider(countryCode)
	config := PhotographyMapConfig{Provider: provider}
	if provider == "amap" {
		config.BrowserKey, config.SecurityKey = secrets.AMapWebKey, secrets.AMapSecurityKey
		config.Configured = config.BrowserKey != ""
		if !config.Configured {
			config.Message = "未配置高德地图 Web Key，仍可使用地址搜索和经纬度输入"
		}
	} else {
		config.BrowserKey = secrets.GoogleMapsKey
		config.Configured = config.BrowserKey != ""
		if !config.Configured {
			config.Message = "未配置 Google Maps Browser Key，仍可使用地址搜索和经纬度输入"
		}
	}
	return config
}

func photographyIntegrationProviderConfig(source string) photographyWeatherProviderConfig {
	key, host, enabled := integrationSecretForWeatherSource(source)
	credentialType := "api_key"
	if source == PhotographyWeatherSourceQWeather {
		credentialType = photographyIntegrationSecretsForUse().QWeatherCredentialType
	}
	return photographyWeatherProviderConfig{APIKey: key, APIHost: host, CredentialType: credentialType, Enabled: enabled}
}

func integrationValidationError(message string) error { return fmt.Errorf("%s", message) }
