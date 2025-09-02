package initializers

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnvVarables() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error load .env file")
	}
}
