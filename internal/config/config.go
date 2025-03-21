package config

import (
    "fmt"
    "io/ioutil"
    "sync"

    "gopkg.in/yaml.v2"
)

var (
    config *Config
    once   sync.Once
)

type Config struct {
    Bot      BotConfig      `yaml:"bot"`
    Admin    AdminConfig    `yaml:"admin"`
    Database DatabaseConfig `yaml:"database"`
    Channels ChannelsConfig `yaml:"channels"`
    Payment  PaymentConfig  `yaml:"payment"`
    Tickets  TicketsConfig  `yaml:"tickets"`
    Limits   LimitsConfig   `yaml:"limits"`
}

type BotConfig struct {
    Name  string `yaml:"name"`
}


type AdminConfig struct {
    IDs []string `yaml:"ids"`
}

type DatabaseConfig struct {
    DriveFolderID string `yaml:"drive_folder_id"`
}

type ChannelsConfig struct {
    LotteryProof string `yaml:"lottery_proof"`
    LotteryWin   string `yaml:"lottery_win"`
}

type PaymentConfig struct {
    QRCodeLink string `yaml:"qr_code_link"`
}

type TicketsConfig struct {
    Prices []float64 `yaml:"prices"`
}

type LimitsConfig struct {
    MaxInvalidAttempts int `yaml:"max_invalid_attempts"`
    CommandRateLimit   int `yaml:"command_rate_limit"`
}

func Load(filename string) error {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("error reading config file: %v", err)
    }

    cfg := &Config{}
    if err := yaml.Unmarshal(data, cfg); err != nil {
        return fmt.Errorf("error parsing config file: %v", err)
    }

    once.Do(func() {
        config = cfg
    })

    return nil
}

func Get() *Config {
    return config
}
