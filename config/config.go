package config

import (
	"github.com/joho/godotenv"
	"log"
)

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Erro ao carregar as variáveis de ambiente: %v", err)
		return err
	}

	return nil
}
