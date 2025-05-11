package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env  string     `yaml:"env" env-default:"local"`
	GRPC GRPCConfig `yaml:"grpc"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

func MustLoad() *Config {
	err := godotenv.Load("../.env")
	if err != nil {
		panic("Error loading .env file" + err.Error())
	}

	conf := &Config{}

	conf.Env = os.Getenv("ENV")
	if conf.Env == "" {
		panic("Env is empty")
	}

	port := os.Getenv("PORT")
	val, err := strconv.Atoi(port)

	if err != nil {
		conf.GRPC.Port = 8080
	} else {
		conf.GRPC.Port = val

	}

	timer, err := time.ParseDuration(os.Getenv("TIMEOUT"))
	if err != nil {
		conf.GRPC.Timeout = 10 * time.Second
	} else {
		conf.GRPC.Timeout = timer
	}

	return conf

}
