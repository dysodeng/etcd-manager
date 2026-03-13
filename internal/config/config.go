package config

import "github.com/spf13/viper"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
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
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
