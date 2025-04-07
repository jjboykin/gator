package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	DBUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

func Read() (Config, error) {

	/*
		Export a Read function that reads the JSON file found at ~/.gatorconfig.json and returns a Config struct.
		It should read the file from the HOME directory, then decode the JSON string into a new Config struct.
		Use os.UserHomeDir to get the location of HOME.
	*/

	configPath, err := getConfigFilePath()
	if err != nil {
		log.Print("error getting config path")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var cfgPayload Config
	err = json.Unmarshal(data, &cfgPayload)
	return cfgPayload, err
}

func (cfg *Config) SetUser(user string) error {
	/*
		Export a SetUser method on the Config struct that writes the config struct to the JSON file after setting the current_user_name field.
	*/
	cfg.CurrentUserName = user
	return write(*cfg)
}

func getConfigFilePath() (string, error) {
	dir, err := os.UserHomeDir()
	path := dir + "/" + configFileName
	return path, err
}

func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		log.Print("error loading current config path")
	}
	configPath, err := getConfigFilePath()
	if err != nil {
		log.Print("error getting config path")
	}
	file, err := os.Create(configPath)

	if err != nil {
		panic(err)
	}

	length, err := file.Write(data)
	log.Printf("file of size %d written", length)
	if err != nil {
		panic(err)
	}

	return nil
}
