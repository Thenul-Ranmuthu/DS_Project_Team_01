package initializers

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvVaribles() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	err := godotenv.Load(envFile)
	if err != nil {
		log.Fatalf("Error loading env file %s", envFile)
	}
}
