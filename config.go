package main

import (
	"log"

	"github.com/joho/godotenv"
)

func loadConfig() map[string]string {
	envMap, err := godotenv.Read("config.env")
	if err != nil {
		log.Fatal("Error loading file:", err)
	}

	return envMap
}
