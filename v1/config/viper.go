package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
)

type PoolConfig struct {
	Size   int `mapstructure:"size"`
	Buffer int `mapstructure:"buffer"`
}

type Logger struct {
	Level string `mapstructure:"level"`
}

type Config struct {
	Pool PoolConfig `mapstructure:"worker_pool"`
}

var Conf Config

func Load(confPath string, onChanged func(c Config)) error {
	viper.SetConfigFile(confPath)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("viper read in config: %w", err)
	}

	if err := viper.Unmarshal(&Conf); err != nil {
		return fmt.Errorf("viper unmarshal: %w", err)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		if err := viper.Unmarshal(&Conf); err != nil {
			log.Fatalf("viper unmarshal: %v", err)
		} else {
			if onChanged != nil {
				onChanged(Conf)
			}
		}
	})

	viper.WatchConfig()

	return nil
}
