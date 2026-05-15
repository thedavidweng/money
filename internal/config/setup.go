package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type SetupResult struct {
	ConfigPath    string `json:"config_path"`
	EnvPath       string `json:"env_path"`
	DatabasePath  string `json:"database_path"`
	SecretCreated bool   `json:"secret_created"`
	DBCreated     bool   `json:"db_created"`
}

func Setup(configPath string, profile string, force bool) (SetupResult, error) {
	if err := validateProfile(profile); err != nil {
		return SetupResult{}, err
	}
	if configPath == "" {
		configPath = DefaultConfigPath(profile)
	}
	configPath = expandHome(configPath)
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return SetupResult{}, err
	}

	dir := filepath.Dir(configPath)
	envPath := filepath.Join(dir, ".env")
	dbPath := filepath.Join(dir, "data", "money.db")

	result := SetupResult{
		ConfigPath:   configPath,
		EnvPath:      envPath,
		DatabasePath: dbPath,
	}

	// Create directory
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return result, fmt.Errorf("create config directory: %w", err)
	}

	// Write config.yaml if missing
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		content := configSkeleton(dbPath)
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			return result, fmt.Errorf("write config: %w", err)
		}
	}

	// Generate encryption key and write .env if key is missing
	secretCreated, err := ensureEncryptionKey(envPath, force)
	if err != nil {
		return result, err
	}
	result.SecretCreated = secretCreated

	return result, nil
}

func configSkeleton(dbPath string) string {
	home, _ := os.UserHomeDir()
	rel := dbPath
	if strings.HasPrefix(dbPath, home) {
		rel = "~" + strings.TrimPrefix(dbPath, home)
	}
	return fmt.Sprintf(`database:
  path: %s
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY

providers: {}
`, rel)
}

// ProviderSpec defines the fields needed for a provider configuration.
type ProviderSpec struct {
	Name           string
	SecretFields   []string          // written to .env
	OptionalFields map[string]string // field -> default value, written to config.yaml
	HelpURL        string            // URL where users can obtain API credentials
}

var PlaidSpec = ProviderSpec{
	Name:         "plaid",
	SecretFields: []string{"client_id", "secret"},
	OptionalFields: map[string]string{
		"environment":                    "sandbox",
		"products":                       "transactions",
		"country_codes":                  "US",
		"redirect_uri":                   "",
		"additional_consented_products":  "",
		"required_if_supported_products": "",
		"optional_products":              "",
	},
	HelpURL: "https://dashboard.plaid.com/developers/keys",
}

var BridgeSpec = ProviderSpec{
	Name:         "bridge",
	SecretFields: []string{"client_id", "client_secret"},
	OptionalFields: map[string]string{
		"user_email": "",
	},
	HelpURL: "https://dashboard.bridgeapi.io/dashboard/secret-management",
}

// ProviderSpecByName returns the spec for a known provider.
func ProviderSpecByName(name string) (ProviderSpec, bool) {
	switch name {
	case "plaid":
		return PlaidSpec, true
	case "bridge":
		return BridgeSpec, true
	}
	return ProviderSpec{}, false
}

type ConfigureResult struct {
	Provider    string `json:"provider"`
	EnvPath     string `json:"env_path"`
	ConfigPath  string `json:"config_path"`
	KeysWritten int    `json:"keys_written"`
}

// ConfigureProvider writes provider credentials to .env and env: references to config.yaml.
func ConfigureProvider(configPath string, profile string, spec ProviderSpec, secrets map[string]string, options map[string]string, force bool) (ConfigureResult, error) {
	meta, err := ResolveMetadata(Options{ConfigPath: configPath, Profile: profile})
	if err != nil {
		return ConfigureResult{}, err
	}
	configPath = meta.ConfigPath
	envPath := meta.EnvPath

	result := ConfigureResult{
		Provider:   spec.Name,
		EnvPath:    envPath,
		ConfigPath: configPath,
	}

	// Write secrets to .env
	existing, _ := readDotEnv(envPath)
	envLines := []string{}
	if content, err := os.ReadFile(envPath); err == nil {
		envLines = strings.Split(string(content), "\n")
	}

	keysWritten := 0
	var conflicts []string
	for _, field := range spec.SecretFields {
		envVar := providerEnvVar(spec.Name, field)
		if !force && existing[envVar] != "" {
			conflicts = append(conflicts, envVar)
			continue
		}
		envLines = setEnvLine(envLines, envVar, secrets[field])
		keysWritten++
	}
	if len(conflicts) > 0 {
		return result, fmt.Errorf("env file already contains %s; use --force to overwrite", strings.Join(conflicts, ", "))
	}
	result.KeysWritten = keysWritten

	if err := os.WriteFile(envPath, []byte(strings.Join(envLines, "\n")), 0o600); err != nil {
		return result, fmt.Errorf("write env file: %w", err)
	}

	// Update config.yaml with env: references and optional fields
	if err := updateProviderConfig(configPath, spec, options); err != nil {
		return result, err
	}

	return result, nil
}

func ProviderCredentialConflicts(configPath string, profile string, spec ProviderSpec) ([]string, string, error) {
	meta, err := ResolveMetadata(Options{ConfigPath: configPath, Profile: profile})
	if err != nil {
		return nil, "", err
	}
	existing, _ := readDotEnv(meta.EnvPath)
	var conflicts []string
	for _, field := range spec.SecretFields {
		envVar := providerEnvVar(spec.Name, field)
		if existing[envVar] != "" {
			conflicts = append(conflicts, envVar)
		}
	}
	return conflicts, meta.EnvPath, nil
}

func providerEnvVar(provider, field string) string {
	return strings.ToUpper(provider) + "_" + strings.ToUpper(field)
}

func setEnvLine(lines []string, key, value string) []string {
	prefix := key + "="
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = prefix + value
			return lines
		}
	}
	// Append
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = append(lines[:len(lines)-1], prefix+value, "")
	} else {
		lines = append(lines, prefix+value)
	}
	return lines
}

func updateProviderConfig(configPath string, spec ProviderSpec, options map[string]string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config for update: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	providers, _ := raw["providers"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
		raw["providers"] = providers
	}

	providerBlock := map[string]any{}
	for _, field := range spec.SecretFields {
		providerBlock[field] = map[string]any{"env": providerEnvVar(spec.Name, field)}
	}
	for field, defaultVal := range spec.OptionalFields {
		val := options[field]
		if val == "" {
			val = defaultVal
		}
		if val != "" {
			providerBlock[field] = val
		}
	}
	// Preserve existing fields not managed by this update (e.g. consent
	// product fields configured outside the login flow). Without this,
	// updateProviderConfig silently drops them when building a fresh block.
	if existing, ok := providers[spec.Name].(map[string]any); ok {
		for k, v := range existing {
			if _, exists := providerBlock[k]; !exists {
				providerBlock[k] = v
			}
		}
	}

	providers[spec.Name] = providerBlock

	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0o644)
}

func ensureEncryptionKey(envPath string, force bool) (bool, error) {
	existing, _ := readDotEnv(envPath)
	if !force && existing["MONEY_DB_ENCRYPTION_KEY"] != "" {
		return false, nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return false, fmt.Errorf("generate encryption key: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(key)

	lines := []string{}
	if _, err := os.Stat(envPath); err == nil {
		content, err := os.ReadFile(envPath)
		if err != nil {
			return false, fmt.Errorf("read env file: %w", err)
		}
		replaced := false
		for _, line := range strings.Split(string(content), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "MONEY_DB_ENCRYPTION_KEY=") {
				lines = append(lines, "MONEY_DB_ENCRYPTION_KEY="+encoded)
				replaced = true
			} else {
				lines = append(lines, line)
			}
		}
		if !replaced {
			if len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "MONEY_DB_ENCRYPTION_KEY="+encoded)
		}
	} else {
		lines = []string{"MONEY_DB_ENCRYPTION_KEY=" + encoded, ""}
	}

	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return false, fmt.Errorf("write env file: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(envPath, 0o600)
	}
	return true, nil
}
