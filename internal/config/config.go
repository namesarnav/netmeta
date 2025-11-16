package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	BGP   BGPConfig   `mapstructure:"bgp"`
	OSPF  OSPFConfig  `mapstructure:"ospf"`
	MPLS  MPLSConfig  `mapstructure:"mpls"`
	Auto  AutoConfig  `mapstructure:"auto"`
	API   APIConfig   `mapstructure:"api"`
	DB    DBConfig    `mapstructure:"db"`
}

type BGPConfig struct {
	Peers []BGPPeer `mapstructure:"peers"`
}

type BGPPeer struct {
	Address string `mapstructure:"address"`
	ASN     uint32 `mapstructure:"asn"`
	Port    uint16 `mapstructure:"port"`
}

type OSPFConfig struct {
	Interface string `mapstructure:"interface"`
	PCAPFile  string `mapstructure:"pcap_file"`
}

type MPLSConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type AutoConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	FlapThreshold  int  `mapstructure:"flap_threshold"`
	FlapWindowSec  int  `mapstructure:"flap_window_sec"`
}

type APIConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DBConfig struct {
	Path string `mapstructure:"path"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/netmeta")

	// Set defaults
	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("db.path", "/var/lib/netmeta")
	viper.SetDefault("auto.enabled", true)
	viper.SetDefault("auto.flap_threshold", 3)
	viper.SetDefault("auto.flap_window_sec", 300)
	viper.SetDefault("mpls.enabled", true)

	// Environment variables
	viper.SetEnvPrefix("NETMETA")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Ensure DB directory exists
	if cfg.DB.Path != "" {
		if err := os.MkdirAll(cfg.DB.Path, 0755); err != nil {
			return nil, fmt.Errorf("error creating DB directory: %w", err)
		}
	}

	return &cfg, nil
}

