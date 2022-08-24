package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Redis RedisConfig `json:"redis"`
}

type RedisConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

var Redis RedisConfig

func Init() {
	bytes, err := os.ReadFile("./config.json")
	if err != nil {
		log.Println(err)
	}

	config := Config{}
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		log.Println(err)
	}

	Redis = config.Redis
}
