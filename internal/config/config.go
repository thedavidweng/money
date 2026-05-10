package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/thedavidweng/money/internal/contracts"
)

type Options struct {
	ConfigPath string
	Env        map[string]string
}

type Config struct {
	ConfigPath            string
	EnvPath               string
	DatabasePath          string
	DatabaseEncryptionKey string
	DatabaseEncryptionKeyBytes []byte
	ReadOnly              bool
	Providers             map[string]ProviderConfig
	Warnings              []contracts.Warning
}

type ProviderConfig struct {
	Fields map[string]string
}

type rawConfig struct {
	EnvFile   string                         `yaml:"env_file"`
	ReadOnly  bool                           `yaml:"read_only"`
	Database  rawDatabase                    `yaml:"database"`
	Providers map[string]map[string]yamlNode `yaml:"providers"`
}

type rawDatabase struct {
	Path          string   `yaml:"path"`
	EncryptionKey yamlNode `yaml:"encryption_key"`
}

type yamlNode struct {
	Value any
}

func (n *yamlNode) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		n.Value = value.Value
	case yaml.SequenceNode:
		values := make([]string, 0, len(value.Content))
		for _, child := range value.Content {
			values = append(values, child.Value)
		}
		n.Value = values
	case yaml.MappingNode:
		if len(value.Content) == 2 && value.Content[0].Value == "env" {
			n.Value = envReference{Name: value.Content[1].Value}
			return nil
		}
		return fmt.Errorf("unsupported YAML object; secret references must be {env: NAME}")
	default:
		return fmt.Errorf("unsupported YAML node kind %d", value.Kind)
	}
	return nil
}

type envReference struct {
	Name string
}

func Load(options Options) (Config, error) {
	configPath := options.ConfigPath
	if configPath == "" {
		configPath = options.Env["MONEY_CONFIG"]
	}
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		configPath = filepath.Join(home, ".money", "config.yaml")
	}
	configPath = expandHome(configPath)
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return Config{}, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", configPath, err)
	}
	var raw rawConfig
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return Config{}, err
	}

	envPath := raw.EnvFile
	if envPath == "" {
		envPath = filepath.Join(filepath.Dir(configPath), ".env")
	} else if !filepath.IsAbs(envPath) {
		envPath = filepath.Join(filepath.Dir(configPath), envPath)
	}

	mergedEnv := map[string]string{}
	fileEnv, err := readDotEnv(envPath)
	if err != nil {
		return Config{}, err
	}
	for key, value := range fileEnv {
		mergedEnv[key] = value
	}
	if options.Env == nil {
		options.Env = processEnv()
	}
	for key, value := range options.Env {
		mergedEnv[key] = value
	}

	cfg := Config{
		ConfigPath: configPath,
		EnvPath:    envPath,
		ReadOnly:   raw.ReadOnly || mergedEnv["MONEY_READ_ONLY"] == "1",
		Providers:  map[string]ProviderConfig{},
	}
	if raw.Database.Path == "" {
		return Config{}, fmt.Errorf("database.path is required")
	}
	cfg.DatabasePath = resolvePath(raw.Database.Path, filepath.Dir(configPath))
	key, warnings, err := resolveValue("database.encryption_key", raw.Database.EncryptionKey, mergedEnv, true)
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseEncryptionKey = key
	keyBytes, err := decodeDatabaseKey(key)
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseEncryptionKeyBytes = keyBytes
	cfg.Warnings = append(cfg.Warnings, warnings...)

	for provider, fields := range raw.Providers {
		resolved := ProviderConfig{Fields: map[string]string{}}
		for name, node := range fields {
			value, warnings, err := resolveValue("providers."+provider+"."+name, node, mergedEnv, isSecretField(name))
			if err != nil {
				return Config{}, err
			}
			cfg.Warnings = append(cfg.Warnings, warnings...)
			resolved.Fields[name] = value
		}
		cfg.Providers[provider] = resolved
	}

	return cfg, nil
}

func decodeDatabaseKey(value string) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("database encryption key must be base64url without padding")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("database encryption key must decode to 32 bytes")
	}
	return key, nil
}

func resolveValue(path string, node yamlNode, env map[string]string, secret bool) (string, []contracts.Warning, error) {
	switch value := node.Value.(type) {
	case envReference:
		resolved, ok := env[value.Name]
		if !ok || resolved == "" {
			return "", nil, fmt.Errorf("%s references missing environment variable %s", path, value.Name)
		}
		return resolved, nil, nil
	case string:
		if secret && value != "" {
			return value, []contracts.Warning{{
				Code:     "DIRECT_SECRET_IN_CONFIG",
				Message:  path + " stores a direct secret; use an env reference.",
				Category: contracts.CategoryConfig,
			}}, nil
		}
		return value, nil, nil
	case []string:
		return strings.Join(value, ","), nil, nil
	case nil:
		return "", nil, nil
	default:
		return "", nil, fmt.Errorf("%s has unsupported value type %T", path, value)
	}
}

func readDotEnv(path string) (map[string]string, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env line in %s", path)
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return values, scanner.Err()
}

func processEnv() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env
}

func resolvePath(path string, baseDir string) string {
	path = expandHome(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func isSecretField(name string) bool {
	switch name {
	case "secret", "client_secret", "api_key", "partner_secret", "encryption_key":
		return true
	default:
		return false
	}
}
