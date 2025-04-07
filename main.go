package main

import (
	"fmt"

	"github.com/jjboykin/gator/internal/config"
)

func main() {

	cfg, err := config.Read()
	if err != nil {
		return
	}
	cfg.SetUser("jjboykin")
	fmt.Printf("DB URL: %v\n", cfg.DBUrl)
	fmt.Printf("Username: %v\n", cfg.CurrentUserName)

}
