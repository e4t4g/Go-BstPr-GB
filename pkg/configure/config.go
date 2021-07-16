package configure

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
)

var (
	jsonFile = flag.String("jsonFile", "tsconfig.json", "add link to json config file")
)

type Config struct {
	Url string `json:"url"`
}

func CreateNew() (*Config, error) {

	var config *Config

	if *jsonFile == "" {
		config = &Config{
			"",
		}
	} else {
		confJsonFile, err := ioutil.ReadFile(*jsonFile)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(confJsonFile, &config)
		if err != nil {
			log.Println(err)
		}

	}

	_, err := url.ParseRequestURI(config.Url)
	if err != nil {
		return nil, err
	}

	return config, nil
}
