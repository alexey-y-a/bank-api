package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/sirupsen/logrus"
)

const (
	envConfigFileName = ".env"
)

type Config struct {
	env *EnvSetting
}

type EnvSetting struct {
	ServerHost string `env:"SERVER_HOST" env-default:"0.0.0.0" env-description:"HTTP server host"`
	ServerPort int    `env:"SERVER_PORT" env-default:"8080" env-description:"HTTP server port"`

	// Логирование
	LogLevel  string `env:"LOG_LEVEL" env-default:"info" env-description:"log level: debug, info, warn, error"`
	LogFormat string `env:"LOG_FORMAT" env-default:"text" env-description:"log format: text or json"`

	// База данных
	DBDSN             string `env:"DB_DSN" env-required:"true" env-description:"PostgreSQL connection string"`
	DBMaxOpenConns    int    `env:"DB_MAX_OPEN_CONNS" env-default:"25" env-description:"max open DB connections"`
	DBMaxIdleConns    int    `env:"DB_MAX_IDLE_CONNS" env-default:"5" env-description:"max idle DB connections"`
	DBConnMaxLifetime int    `env:"DB_CONN_MAX_LIFETIME" env-default:"3600" env-description:"connection max lifetime in seconds"`

	// JWT
	JWTSecret   string `env:"JWT_SECRET" env-required:"true" env-description:"JWT signing secret key"`
	JWTTTLHours int    `env:"JWT_TTL_HOURS" env-default:"24" env-description:"JWT token TTL in hours"`

	// Шифрование карт
	CardSecret string `env:"CARD_SECRET" env-required:"true" env-description:"HMAC secret for card data encryption"`

	// SMTP
	SMTPHost     string `env:"SMTP_HOST" env-default:"smtp.example.com" env-description:"SMTP server host"`
	SMTPPort     int    `env:"SMTP_PORT" env-default:"587" env-description:"SMTP server port"`
	SMTPUser     string `env:"SMTP_USER" env-default:"" env-description:"SMTP username"`
	SMTPPass     string `env:"SMTP_PASS" env-default:"" env-description:"SMTP password"`
	SMTPFrom     string `env:"SMTP_FROM" env-default:"noreply@example.com" env-description:"SMTP sender email"`
	SMTPNotifyTo string `env:"SMTP_NOTIFY_TO" env-default:"" env-description:"email for test notifications"`
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

func issetEnvConfigFile() bool {
	_, err := os.Stat(envConfigFileName)
	return err == nil
}

func (e *EnvSetting) GetHelpString() (string, error) {
	customHeader := "Environment variables configuration:"
	helpString, err := cleanenv.GetDescription(e, &customHeader)
	if err != nil {
		return "", fmt.Errorf("get help string failed: %w", err)
	}
	return helpString, nil
}

func New() *Config {
	envSetting := &EnvSetting{}

	helpString, err := envSetting.GetHelpString()
	if err != nil {
		logrus.Panic("getting help string of env settings failed: ", err)
	}
	logrus.Info(helpString)

	if issetEnvConfigFile() {
		err := cleanenv.ReadConfig(envConfigFileName, envSetting)
		if err != nil {
			logrus.Panicf("read env config file failed: %s", err)
		}
	} else if err := cleanenv.ReadEnv(envSetting); err != nil {
		logrus.Panicf("read env config failed: %s", err)
	}

	return &Config{env: envSetting}
}

func (c *Config) PrintDebug() {
	envReflect := reflect.Indirect(reflect.ValueOf(c.env))
	envReflectType := envReflect.Type()

	exp := regexp.MustCompile(`(?i)(token|password|secret|pass)`)

	for i := range envReflect.NumField() {
		key := envReflectType.Field(i).Name

		if exp.MatchString(key) {
			val, _ := envReflect.Field(i).Interface().(string)
			logrus.Debugf("%s: len=%d (masked)", key, len(val))
			continue
		}

		logrus.Debugf("%s: %v", key, envReflect.Field(i).Interface())
	}
}

func (c *Config) GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		DSN:             c.env.DBDSN,
		MaxOpenConns:    c.env.DBMaxOpenConns,
		MaxIdleConns:    c.env.DBMaxIdleConns,
		ConnMaxLifetime: c.env.DBConnMaxLifetime,
	}
}

func (c *Config) GetServerHost() string {
	return c.env.ServerHost
}

func (c *Config) GetServerPort() int {
	return c.env.ServerPort
}

func (c *Config) GetLogLevel() string {
	return c.env.LogLevel
}

func (c *Config) GetLogFormat() string {
	return c.env.LogFormat
}

func (c *Config) GetDBDSN() string {
	return c.env.DBDSN
}

func (c *Config) GetDBMaxOpenConns() int {
	return c.env.DBMaxOpenConns
}

func (c *Config) GetDBMaxIdleConns() int {
	return c.env.DBMaxIdleConns
}

func (c *Config) GetDBConnMaxLifetime() int {
	return c.env.DBConnMaxLifetime
}

func (c *Config) GetJWTSecret() string {
	return c.env.JWTSecret
}

func (c *Config) GetJWTTTLHours() int {
	return c.env.JWTTTLHours
}

func (c *Config) GetCardSecret() string {
	return c.env.CardSecret
}

func (c *Config) GetSMTPHost() string {
	return c.env.SMTPHost
}

func (c *Config) GetSMTPPort() int {
	return c.env.SMTPPort
}

func (c *Config) GetSMTPUser() string {
	return c.env.SMTPUser
}

func (c *Config) GetSMTPPass() string {
	return c.env.SMTPPass
}

func (c *Config) GetSMTPFrom() string {
	return c.env.SMTPFrom
}

func (c *Config) GetSMTPNotifyTo() string {
	return c.env.SMTPNotifyTo
}
