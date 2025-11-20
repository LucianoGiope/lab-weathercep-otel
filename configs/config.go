package configs

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var cfg *conf

type conf struct {
	UrlWeather                         string `mapstructure:"URLWEATHER"`
	APIKeyWeather                      string `mapstructure:"APIKEYWEATHER"`
	UrlCep                             string `mapstructure:"URLCEP"`
	UrlServerWeather                   string `mapstructure:"URLSERVERWEATHER"`
	Otel_Exporter_Otlp_Traces_Endpoint string `mapstructure:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"`
	Otel_Exporter_Otlp_Protocol        string `mapstructure:"OTEL_EXPORTER_OTLP_PROTOCOL"`
	Otel_Service_Name                  string `mapstructure:"OTEL_SERVICE_NAME"`
}

func LoadConfig(path string) (*conf, error) {
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	for _, key := range viper.AllKeys() {
		if strings.TrimSpace(viper.GetString(key)) == "" {
			return nil, fmt.Errorf("variável de ambiente: %s não foi informada", key)
		}
	}
	return cfg, nil
}
