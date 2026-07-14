package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port              int           `mapstructure:"port"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

type EtcdConfig struct {
	Endpoints []string  `mapstructure:"endpoints"`
	Username  string    `mapstructure:"username"`
	Password  string    `mapstructure:"password"`
	TLS       TLSConfig `mapstructure:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"` // sqlite 或 postgres
	Path   string `mapstructure:"path"`   // SQLite 文件路径
	DSN    string `mapstructure:"dsn"`    // PostgreSQL 连接串
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetDefault("server.read_header_timeout", "5s")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.shutdown_timeout", "15s")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("etcd.endpoints", "ETCD_ENDPOINTS")
	_ = v.BindEnv("etcd.username", "ETCD_USERNAME")
	_ = v.BindEnv("etcd.password", "ETCD_PASSWORD")
	_ = v.BindEnv("database.driver", "DB_DRIVER")
	_ = v.BindEnv("database.dsn", "DB_DSN")
	_ = v.BindEnv("server.read_header_timeout", "SERVER_READ_HEADER_TIMEOUT")
	_ = v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	_ = v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	_ = v.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")
	_ = v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")
	_ = v.BindEnv("log.level", "LOG_LEVEL")
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
