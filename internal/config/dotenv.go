package config

import (
	"errors"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

// LoadDotEnv carrega o arquivo .env, se existir. Variáveis já presentes no
// ambiente vencem o arquivo — é assim que produção sobrescreve o local.
func LoadDotEnv(path string) error {
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
