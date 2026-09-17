package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"
	"golang.org/x/crypto/bcrypt"

	"github.com/kgretzky/pwndrop/log"
	"github.com/kgretzky/pwndrop/storage"
	"github.com/kgretzky/pwndrop/utils"
)

const (
	INI_SERVER         = "pwndrop"
	INI_VAR_LISTEN_IP  = "listen_ip"
	INI_VAR_HTTP_PORT  = "http_port"
	INI_VAR_HTTPS_PORT = "https_port"
	INI_VAR_DATA_DIR   = "data_dir"
	INI_VAR_ADMIN_DIR  = "admin_dir"

	INI_SETUP              = "setup"
	INI_SETUP_USERNAME     = "username"
	INI_SETUP_PASSWORD     = "password"
	INI_SETUP_REDIRECT_URL = "redirect_url"
	INI_SETUP_SECRET_PATH  = "secret_path"
)

type serverConfig struct {
	ListenIP  string `toml:"listen_ip"`
	HTTPPort  int    `toml:"http_port"`
	HTTPSPort int    `toml:"https_port"`
	DataDir   string `toml:"data_dir"`
	AdminDir  string `toml:"admin_dir"`
}

type setupConfig struct {
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	RedirectURL string `toml:"redirect_url"`
	SecretPath  string `toml:"secret_path"`
}

type fileConfig struct {
	Pwndrop serverConfig `toml:"pwndrop"`
	Setup   *setupConfig `toml:"setup,omitempty"`
}

type Config struct {
	file    fileConfig
	path    string
	execDir string
	dirty   bool
}

func NewConfig(path string) (*Config, error) {
	c := &Config{
		path:    path,
		execDir: utils.GetExecDir(),
		file: fileConfig{Pwndrop: serverConfig{
			ListenIP:  "",
			HTTPPort:  80,
			HTTPSPort: 443,
			DataDir:   filepath.Join(utils.GetExecDir(), "data"),
			AdminDir:  filepath.Join(utils.GetExecDir(), "admin"),
		}},
	}

	tree, err := toml.LoadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read TOML config: %w", err)
		}
		log.Warning("config file not found at path: %s", path)
		c.dirty = true
		return c, nil
	}
	if err := tree.Unmarshal(&c.file); err != nil {
		return nil, fmt.Errorf("parse TOML config: %w", err)
	}
	c.applyDefaults()
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.file.Pwndrop.HTTPPort == 0 {
		c.file.Pwndrop.HTTPPort = 80
	}
	if c.file.Pwndrop.DataDir == "" {
		c.file.Pwndrop.DataDir = filepath.Join(c.execDir, "data")
	}
	if c.file.Pwndrop.AdminDir == "" {
		c.file.Pwndrop.AdminDir = filepath.Join(c.execDir, "admin")
	}
}

// HandleSetup applies one-time TOML bootstrap values, then environment values.
// Environment credentials are never persisted to the TOML file and are only used
// while there are no existing application users.
func (c *Config) HandleSetup() error {
	if c.file.Setup != nil {
		if err := c.applySetup(*c.file.Setup); err != nil {
			return err
		}
		c.file.Setup = nil
		c.dirty = true
	}

	envSetup := setupConfig{
		Username:    os.Getenv("PWN_DROP_SETUP_USERNAME"),
		Password:    os.Getenv("PWN_DROP_SETUP_PASSWORD"),
		RedirectURL: os.Getenv("PWN_DROP_SETUP_REDIRECT_URL"),
		SecretPath:  os.Getenv("PWN_DROP_SETUP_SECRET_PATH"),
	}
	if envSetup.Username != "" || envSetup.Password != "" || envSetup.RedirectURL != "" || envSetup.SecretPath != "" {
		users, err := storage.UserList()
		if err != nil {
			return fmt.Errorf("list users for environment setup: %w", err)
		}
		if len(users) == 0 {
			if err := c.applySetup(envSetup); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Config) applySetup(setup setupConfig) error {
	o, err := storage.ConfigGet(1)
	if err != nil {
		return fmt.Errorf("get database config: %w", err)
	}

	if setup.RedirectURL != "" {
		o.RedirectUrl = setup.RedirectURL
		log.Important("setup: redirect URL configured")
	}
	if setup.SecretPath != "" {
		secretPath := setup.SecretPath
		if !strings.HasPrefix(secretPath, "/") {
			secretPath = "/" + secretPath
		}
		if len(secretPath) > 1 {
			o.CookieName = utils.GenRandomString(4)
			o.CookieToken = utils.GenRandomHash()
			o.SecretPath = secretPath
			log.Important("setup: secret path configured")
		}
	}

	users, err := storage.UserList()
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	if len(users) == 0 && setup.Username != "" && setup.Password != "" {
		phash, err := bcrypt.GenerateFromPassword([]byte(setup.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash setup password: %w", err)
		}
		if _, err := storage.UserCreate(&storage.DbUser{Name: setup.Username, Password: string(phash)}); err != nil {
			return fmt.Errorf("create setup user: %w", err)
		}
		log.Important("setup: created initial administrator account")
	}

	if _, err := storage.ConfigUpdate(1, o); err != nil {
		return fmt.Errorf("save database config: %w", err)
	}
	return nil
}

func (c *Config) Save() error {
	if !c.dirty {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	encoded, err := toml.Marshal(c.file)
	if err != nil {
		return fmt.Errorf("encode TOML config: %w", err)
	}
	if err := os.WriteFile(c.path, encoded, 0600); err != nil {
		return fmt.Errorf("save TOML config: %w", err)
	}
	c.dirty = false
	return nil
}

func (c *Config) GetListenIP() string { return c.file.Pwndrop.ListenIP }
func (c *Config) GetHttpPort() int    { return c.file.Pwndrop.HTTPPort }
func (c *Config) GetHttpsPort() int   { return c.file.Pwndrop.HTTPSPort }
func (c *Config) GetSecretPath() string {
	o, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return o.SecretPath
}
func (c *Config) GetDataDir() string  { return c.joinPath(c.execDir, c.file.Pwndrop.DataDir) }
func (c *Config) GetAdminDir() string { return c.joinPath(c.execDir, c.file.Pwndrop.AdminDir) }
func (c *Config) GetCookieName() string {
	o, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return o.CookieName
}
func (c *Config) GetCookieToken() string {
	o, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return o.CookieToken
}
func (c *Config) GetRedirectUrl() string {
	o, err := storage.ConfigGet(1)
	if err != nil {
		return ""
	}
	return o.RedirectUrl
}

func (c *Config) Get(key string) (string, error) {
	switch key {
	case INI_VAR_LISTEN_IP:
		return c.file.Pwndrop.ListenIP, nil
	case INI_VAR_HTTP_PORT:
		return strconv.Itoa(c.file.Pwndrop.HTTPPort), nil
	case INI_VAR_HTTPS_PORT:
		return strconv.Itoa(c.file.Pwndrop.HTTPSPort), nil
	case INI_VAR_DATA_DIR:
		return c.file.Pwndrop.DataDir, nil
	case INI_VAR_ADMIN_DIR:
		return c.file.Pwndrop.AdminDir, nil
	default:
		return "", fmt.Errorf("config key %q not found", key)
	}
}

func (c *Config) Set(key string, value string) error {
	switch key {
	case INI_VAR_LISTEN_IP:
		c.file.Pwndrop.ListenIP = value
	case INI_VAR_HTTP_PORT:
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HTTP port: %w", err)
		}
		c.file.Pwndrop.HTTPPort = port
	case INI_VAR_HTTPS_PORT:
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HTTPS port: %w", err)
		}
		c.file.Pwndrop.HTTPSPort = port
	case INI_VAR_DATA_DIR:
		c.file.Pwndrop.DataDir = value
	case INI_VAR_ADMIN_DIR:
		c.file.Pwndrop.AdminDir = value
	default:
		return fmt.Errorf("config key %q not found", key)
	}
	c.dirty = true
	return c.Save()
}

func (c *Config) joinPath(basePath string, relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(basePath, relPath)
}
