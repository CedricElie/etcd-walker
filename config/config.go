package config

import (
	"log"
	"sync"
	"github.com/spf13/viper"
)

type Config struct {
	ETCD_HOST    string `mapstructure:"ETCD_HOST"`
	TestDataPath string `mapstructure:"testDataPath"`
}

var (
    cfg  *Config
    once sync.Once
)

func LoadConfig(configPath string, configType string) (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType(configType)
    viper.AddConfigPath(configPath)

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }


	return &cfg, nil
}

func GetConfig() *Config {
    once.Do(func() {
        loadedCfg, err := LoadConfig("./config/", "yaml") // Adjust path and type as needed
        if err != nil {
            log.Fatalf("Failed to load config: %v", err)
        }
        cfg = loadedCfg
    })
    return cfg
}