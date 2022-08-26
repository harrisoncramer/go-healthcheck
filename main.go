package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	yaml "github.com/go-yaml/yaml"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

type Config struct {
	Schedule int
}

var (
	ErrorScheduleNotSet = "Configuration file error: Schedule must be set."
)

func validateConfig(config Config) error {
	log.Println(config.Schedule)
	if config.Schedule == 0 {
		return errors.New(ErrorScheduleNotSet)
	}
	return nil
}

func main() {

	log.Println("Starting...")
	config := Config{}

	settingsPath := os.Args[1]

	data, err := os.ReadFile(settingsPath)
	check(err)

	err = yaml.Unmarshal(data, &config)
	check(err)

	err = validateConfig(config)
	check(err)

	s := gocron.NewScheduler(time.UTC)
	s.Every(config.Schedule).Milliseconds().Do(func() {
		log.Printf("Hello World!")
	})

	s.StartBlocking()
}
