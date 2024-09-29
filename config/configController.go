package config

import (
	"github.com/spf13/viper"
)

func InitConfig() {
	viper.AddConfigPath("./")
	viper.SetConfigName("config") // Register config file name (no extension)
	viper.SetConfigType("json")   // Look for specific type
	viper.ReadInConfig()
}

func GetFileServiceURL(endpoint string) string {
	baseURL := viper.GetString("file_service.base_url")
	endpointPath := viper.GetString("file_service.endpoints." + endpoint)
	return baseURL + endpointPath
}
