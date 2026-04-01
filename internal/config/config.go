package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the runtime configuration used by the application.
type Config struct {
	Addr           string
	MySQLDSN       string
	JWTSecret      string
	UploadDir      string
	StaticURLPath  string
	AdminUser      string
	AdminPass      string
	BidRateBurst   int
	BidRateEvery   time.Duration
	PaymentWebhook string
}

type yamlFile struct {
	Addr                string `yaml:"addr"`
	MySQLDSN            string `yaml:"mysql_dsn"`
	JWTSecret           string `yaml:"jwt_secret"`
	UploadDir           string `yaml:"upload_dir"`
	StaticURLPath       string `yaml:"static_url"`
	AdminUser           string `yaml:"admin_user"`
	AdminPass           string `yaml:"admin_pass"`
	BidRateBurst        int    `yaml:"bid_rate_burst"`
	BidRateEveryMS      int64  `yaml:"bid_rate_every_ms"`
	PaymentWebhook      string `yaml:"payment_webhook_secret"`
}

// Load reads the config file path from -config flag (set in main), env AUCTION_CONFIG, or default "config.yaml".
// Call LoadWithPath from main after parsing flags.
func LoadWithPath(path string) (Config, error) {
	if path == "" {
		path = os.Getenv("AUCTION_CONFIG")
	}
	if path == "" {
		path = "config.yaml"
	}
	return LoadFile(path)
}

// LoadFile parses a YAML file into Config.
func LoadFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var y yamlFile
	if err := yaml.Unmarshal(raw, &y); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	c := Config{
		Addr:           y.Addr,
		MySQLDSN:       y.MySQLDSN,
		JWTSecret:      y.JWTSecret,
		UploadDir:      y.UploadDir,
		StaticURLPath:  y.StaticURLPath,
		AdminUser:      y.AdminUser,
		AdminPass:      y.AdminPass,
		BidRateBurst:   y.BidRateBurst,
		PaymentWebhook: y.PaymentWebhook,
	}
	if y.BidRateEveryMS > 0 {
		c.BidRateEvery = time.Duration(y.BidRateEveryMS) * time.Millisecond
	}
	applyDefaults(&c)
	if err := validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func applyDefaults(c *Config) {
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if c.UploadDir == "" {
		c.UploadDir = "./uploads"
	}
	if c.StaticURLPath == "" {
		c.StaticURLPath = "/static/uploads"
	}
	if c.AdminUser == "" {
		c.AdminUser = "admin"
	}
	if c.AdminPass == "" {
		c.AdminPass = "admin123"
	}
	if c.BidRateBurst <= 0 {
		c.BidRateBurst = 5
	}
	if c.BidRateEvery <= 0 {
		c.BidRateEvery = 800 * time.Millisecond
	}
}

func validate(c Config) error {
	if c.MySQLDSN == "" {
		return errors.New("config: mysql_dsn is required")
	}
	if c.JWTSecret == "" {
		return errors.New("config: jwt_secret is required")
	}
	return nil
}
