package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/kgretzky/pwndrop/log"
	"github.com/kgretzky/pwndrop/storage"
	"github.com/kgretzky/pwndrop/utils"
)

const (
	EnvListenIP  = "PWN_DROP_LISTEN_IP"
	EnvHTTPPort  = "PWN_DROP_HTTP_PORT"
	EnvHTTPSPort = "PWN_DROP_HTTPS_PORT"
	EnvDataDir   = "PWN_DROP_DATA_DIR"
	EnvAdminDir  = "PWN_DROP_ADMIN_DIR"

	EnvSetupUsername    = "PWN_DROP_SETUP_USERNAME"
	EnvSetupPassword    = "PWN_DROP_SETUP_PASSWORD"
	EnvSetupRedirectURL = "PWN_DROP_SETUP_REDIRECT_URL"
	EnvSetupSecretPath  = "PWN_DROP_SETUP_SECRET_PATH"
)

type Config struct {
	listenIP  string
	httpPort  int
	httpsPort int
	dataDir   string
	adminDir  string
	execDir   string
}

type setupConfig struct {
	username    string
	password    string
	redirectURL string
	secretPath  string
}

// NewConfig reads all runtime settings from environment variables. Docker
// Compose supplies them from .env; no configuration file is read or created.
func NewConfig() (*Config, error) {
	execDir := utils.GetExecDir()
	httpPort, err := getPort(EnvHTTPPort, 8080, false)
	if err != nil {
		return nil, err
	}
	httpsPort, err := getPort(EnvHTTPSPort, 0, true)
	if err != nil {
		return nil, err
	}

	return &Config{
		listenIP:  envOrDefault(EnvListenIP, ""),
		httpPort:  httpPort,
		httpsPort: httpsPort,
		dataDir:   envOrDefault(EnvDataDir, filepath.Join(execDir, "data")),
		adminDir:  envOrDefault(EnvAdminDir, filepath.Join(execDir, "admin")),
		execDir:   execDir,
	}, nil
}

func getPort(key string, fallback int, allowZero bool) (int, error) {
	value := envOrDefault(key, strconv.Itoa(fallback))
	port, err := strconv.Atoi(value)
	if err != nil || port < 0 || port > 65535 || (!allowZero && port == 0) {
		return 0, fmt.Errorf("%s must be a valid port number", key)
	}
	return port, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// HandleSetup initializes database-backed settings and the first administrator
// once. Bootstrap variables are ignored after any account exists.
func (c *Config) HandleSetup() error {
	setup := setupConfig{
		username:    os.Getenv(EnvSetupUsername),
		password:    os.Getenv(EnvSetupPassword),
		redirectURL: os.Getenv(EnvSetupRedirectURL),
		secretPath:  os.Getenv(EnvSetupSecretPath),
	}
	if setup.username == "" && setup.password == "" && setup.redirectURL == "" && setup.secretPath == "" {
		return nil
	}

	users, err := storage.UserList()
	if err != nil {
		return fmt.Errorf("list users for environment setup: %w", err)
	}
	if len(users) > 0 {
		return nil
	}
	return c.applySetup(setup)
}

func (c *Config) applySetup(setup setupConfig) error {
	dbConfig, err := storage.ConfigGet(1)
	if err != nil {
		return fmt.Errorf("get database config: %w", err)
	}

	if setup.redirectURL != "" {
		dbConfig.RedirectUrl = setup.redirectURL
		log.Important("setup: redirect URL configured")
	}
	if setup.secretPath != "" {
		secretPath := setup.secretPath
		if !strings.HasPrefix(secretPath, "/") {
			secretPath = "/" + secretPath
		}
		if len(secretPath) > 1 {
			dbConfig.CookieName = utils.GenRandomString(4)
			dbConfig.CookieToken = utils.GenRandomHash()
			dbConfig.SecretPath = secretPath
			log.Important("setup: secret path configured")
		}
	}

	if setup.username != "" && setup.password != "" {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(setup.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash setup password: %w", err)
		}
		if _, err := storage.UserCreate(&storage.DbUser{Name: setup.username, Password: string(passwordHash)}); err != nil {
			return fmt.Errorf("create setup user: %w", err)
		}
		log.Important("setup: created initial administrator account")
	}

	if _, err := storage.ConfigUpdate(1, dbConfig); err != nil {
		return fmt.Errorf("save database config: %w", err)
	}
	return nil
}

func (c *Config) GetListenIP() string { return c.listenIP }
func (c *Config) GetHttpPort() int    { return c.httpPort }
func (c *Config) GetHttpsPort() int   { return c.httpsPort }
func (c *Config) GetDataDir() string  { return c.joinPath(c.dataDir) }
func (c *Config) GetAdminDir() string { return c.joinPath(c.adminDir) }

func (c *Config) GetSecretPath() string {
	dbConfig, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return dbConfig.SecretPath
}

func (c *Config) GetCookieName() string {
	dbConfig, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return dbConfig.CookieName
}

func (c *Config) GetCookieToken() string {
	dbConfig, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return dbConfig.CookieToken
}

func (c *Config) GetRedirectUrl() string {
	dbConfig, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return dbConfig.RedirectUrl
}

func (c *Config) joinPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.execDir, path)
}
