package main

import (
	"errors"
	"flag"
	"fmt"
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
    go-healthcheck -f config_file.yml
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

const colorRed = "\033[31m"
const colorYellow = "\033[33m"
const colorWhite = "\033[37m"

func check(e error) {
	if e != nil {
		log.Print(usage)
		log.Fatal(string(colorRed), e)
	}
}

type Job struct {
	Endpoint string
}

type Config struct {
	Schedule int
	Base_url string
	Port     int
	Jobs     []Job
}

func (o *Config) Init() {
	if o.Base_url == "" {
		log.Println(string(colorYellow), "config warning: config.base_url not set, defaulting to 'http://localhost'")
		o.Base_url = "http://localhost"
	}

	if o.Port == 0 {
		log.Println(string(colorYellow), "config warning: config.port not set, defaulting to 80")
		o.Port = 80
	}
}

var (
	ErrorConfigNotProvided = "config error: Configuration file not provided"
	ErrorScheduleNotSet    = "config error: config.schedule not set"
	ErrorJobEndpointNotSet = func(i int) string {
		return "config error: config.jobs[" + fmt.Sprintf("%v", i) + "] endpoint not set"
	}
)

func validateConfig(config Config) error {
	if config.Schedule == 0 {
		return errors.New(ErrorScheduleNotSet)
	}

	for i, job := range config.Jobs {
		if job.Endpoint == "" {
			return errors.New(ErrorJobEndpointNotSet(i))
		}
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
		log.Println(string(colorWhite), "Running job...")
	})

	s.StartBlocking()
}
