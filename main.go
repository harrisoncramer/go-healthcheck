package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron"
	yaml "github.com/go-yaml/yaml"
)

const usage = `
****************************************************************
                        go-healthcheck
****************************************************************

 SYNOPSIS
    go-healthcheck -f config_file.yml ...
  DESCRIPTION
     This is a script template
     to start any good shell script.
  OPTIONS
     -f [file],                    Set config file. This file will configure 
                                   go-healthcheck, by setting a default base url, 
                                   schedule, and list of checks to perform.
     -v, --version                 Print script information
  EXAMPLES
     go-healthcheck -f production-check.yml
`

func check(e error) {
	colorRed := "\033[31m"
	if e != nil {
		log.Print(usage)
		log.Fatal(string(colorRed), e)
	}
}

type Config struct {
	Schedule int
	Base_url string
}

func (o *Config) Init() {
	if o.Base_url == "" {
		log.Println("config.base_url not set, defaulting to 'http://localhost'")
		o.Base_url = "http://localhost"
	}
}

var (
	ErrorConfigNotProvided = "config error: Configuration file not provided"
	ErrorScheduleNotSet    = "config error: config.schedule not set"
)

func validateConfig(config Config) error {
	if config.Schedule == 0 {
		return errors.New(ErrorScheduleNotSet)
	}

	return nil
}

func main() {
	log.Println("Starting...")

	var config_file = flag.String("f", "", "The configuration file for go-healthcheck")
	flag.Parse()

	if *config_file == "" {
		check(errors.New(ErrorConfigNotProvided))
	}

	config := Config{}

	data, err := os.ReadFile(*config_file)
	check(err)

	err = yaml.Unmarshal(data, &config)
	check(err)

	/* Initialize default values if YAML is unset */
	config.Init()

	err = validateConfig(config)
	check(err)

	s := gocron.NewScheduler(time.UTC)
	s.Every(config.Schedule).Milliseconds().Do(func() {
		log.Printf("Hello World!")
	})

	s.StartBlocking()
}
