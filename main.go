package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

type Json map[string]interface{}

type Job struct {
	Name        string
	Description string
	Endpoint    string
	Status      int
	Body        string
	Read_file   bool
	Json        Json // Added if Read_file is true
}

func (job *Job) addJson(j Json) {
	job.Json = j
}

type Failure struct {
	Job        Job
	StatusCode int
	Body       []byte
	Message    string
}

type Failures struct {
	failures []Failure
}

func (parent *Failures) addFailure(job Job, status int, body []byte, message string) {
	parent.failures = append(parent.failures, Failure{
		Job:        job,
		StatusCode: status,
		Body:       body,
		Message:    message,
	})
}

type Successes struct {
	successes []Job
}

func (parent *Successes) addSuccess(job Job) {
	parent.successes = append(parent.successes, job)
}

type Config struct {
	Schedule int
	Base_url string
	Port     int
	Jobs     []Job
	Verbose  bool
}

/* Sets up some default values if they aren't set in the config.yaml */
func (o *Config) Init() {
	if o.Base_url == "" {
		log.Println(string(colorYellow), "config warning: config.base_url not set, defaulting to 'http://localhost'")
		o.Base_url = "http://localhost"
	}

	if o.Port == 0 {
		log.Println(string(colorYellow), "config warning: config.port not set, defaulting to 80")
		o.Port = 80
	}

	for i, job := range o.Jobs {
		if job.Name == "" {
			o.Jobs[i].Name = "job_" + fmt.Sprintf("%v", i)
		}

		if job.Read_file {
			data, err := os.ReadFile(job.Body)
			check(err)

			var jsonBody Json
			err = json.Unmarshal(data, &jsonBody)
			check(err)

			job.addJson(jsonBody)

		}
	}
}

var (
	ErrorConfigNotProvided = "config error: Configuration file not provided"
	ErrorScheduleNotSet    = "config error: config.schedule not set"
	ErrorJobEndpointNotSet = func(name string) string {
		return "config error: config.jobs[" + fmt.Sprintf("%s", name) + "] endpoint not set"
	}
	ErrorEmptyBody = func(name string) string {
		return fmt.Sprintf("config error: No expected body provided for %s", name)
	}
)

/* Validates that configuration formed via the configuratione file and defaults in the initializer function, if appropriate */
func validateConfig(config Config) error {
	if config.Schedule == 0 {
		return errors.New(ErrorScheduleNotSet)
	}

	for i, job := range config.Jobs {
		jobLogName := job.Name
		if jobLogName == "" {
			jobLogName = fmt.Sprint(i)
		}
		if job.Endpoint == "" {
			return errors.New(ErrorJobEndpointNotSet(jobLogName))
		}

		if job.Body == "" && job.Status != 404 {
			return errors.New(ErrorEmptyBody(jobLogName))
		}
	}

	return nil
}

func checkStatus(job Job, resp *http.Response) bool {
	var expectedStatus int
	if job.Status == 0 {
		expectedStatus = 200
	} else {
		expectedStatus = job.Status
	}
	return resp.StatusCode == expectedStatus
}

func checkBody(job Job, body []byte) bool {
	if string(body) == "" || job.Status == 404 {
		return true
	}

	return job.Body == string(body)
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

	config.Init()

	err = validateConfig(config)
	check(err)

	baseWithPort := config.Base_url + ":" + fmt.Sprintf("%v", config.Port)
	s := gocron.NewScheduler(time.UTC)
	s.Every(config.Schedule).Milliseconds().Do(func() {
		failures := Failures{}
		successes := Successes{}

		for _, job := range config.Jobs {
			log.Println(string(colorWhite), fmt.Sprintf("Running: %s", job.Name))
			if config.Verbose && job.Description != "" {
				log.Println(job.Description)
			}

			resp, err := http.Get(baseWithPort + job.Endpoint)
			check(err)

			body, err := ioutil.ReadAll(resp.Body)
			check(err)

			// Checks the status, then body, in order
			if !checkStatus(job, resp) {
				failures.addFailure(job, resp.StatusCode, body, fmt.Sprintf("%s: Expected %d received %d", job.Name, job.Status, resp.StatusCode))
			} else if !checkBody(job, body) {
				failures.addFailure(job, resp.StatusCode, body, fmt.Sprintf("%s: Response body did not match", job.Name))
			} else {
				successes.addSuccess(job)
			}
		}

		log.Print("\n\n--RESULTS--")
		log.Println(fmt.Sprintf("%d/%d Succeeeded", len(successes.successes), len(config.Jobs)))
		log.Println(fmt.Sprintf("%d/%d Failed", len(failures.failures), len(config.Jobs)))

		for _, failure := range failures.failures {
			log.Print(failure.Message)
			if config.Verbose {
				log.Println(fmt.Sprintf("%s: Expected body was:", failure.Job.Name))
				if failure.Job.Read_file {
					log.Println(&failure.Job.Json)
				} else {
					fmt.Println(string(failure.Job.Body))
				}
				log.Println(fmt.Sprintf("%s: Received body was:", failure.Job.Name))
				fmt.Println(string(failure.Body))
			}
		}
	})

	s.StartBlocking()
}
