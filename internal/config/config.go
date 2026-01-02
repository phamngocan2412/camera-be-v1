package config

import "github.com/spf13/viper"

type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	JWT struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	Database struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"database"`
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")
	viper.AddConfigPath(".")

	viper.AutomaticEnv() // override bằng env vars (rất quan trọng cho production/Docker)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
