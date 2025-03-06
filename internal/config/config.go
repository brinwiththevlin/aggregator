package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const configFileName = ".gatorconfig.json"

func getConfigFilePath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("could not locate your home directory")
	}

	file := dir + "/.gatorconfig.json"
	return file ,nil

}
type Config struct{
	Url string `json:"db_url"`
	Username string `json:"current_user_name"`
}

func Read() (Config, error){
	file, err:= getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	content, err := os.ReadFile(file)

	if err != nil {
		return Config{}, fmt.Errorf("could not read config file at %s", file)
	}

	c := Config{}
	err = json.Unmarshal(content, &c)
	return c, nil
}

func (c *Config) SetUser(username string) error {
	c.Username = username
	data, err:= json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	file, err := getConfigFilePath()
	if err != nil {
		return err
	}
	err = os.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
