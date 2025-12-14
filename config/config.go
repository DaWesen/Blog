package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port     int    `mapstructure:"port"`
	Mode     string `mapstructure:"mode"`
	GrpcPort int    `mapstructure:"grpc_port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// 添加多个可能的配置路径，包括config子目录
	viper.AddConfigPath(cwd)                          // 当前目录
	viper.AddConfigPath(filepath.Join(cwd, "config")) // config子目录
	viper.AddConfigPath(".")                          // 当前目录（相对路径）
	viper.AddConfigPath("./config")                   // config子目录（相对路径）
	viper.AddConfigPath("config")                     // config子目录（相对路径）

	// 设置默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.grpc_port", 50051)
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 返回更详细的错误信息
		return nil, err
	}

	// 读取环境变量（可选）
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
