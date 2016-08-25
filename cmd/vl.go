// Copyright © 2016 Ellison Leão <ellisonleao[at]gmail.com>

package cmd

import (
	"fmt"
	"os"
	"io/ioutil"
	"regexp"
	"github.com/spf13/cobra"
	"net/http"
	"time"
	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
)

var Debug bool
var Doc string

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()

type HttpResponse struct {
	url	string
	response *http.Response
	err	error
}

var RootCmd = &cobra.Command{
	Use:   "vl",
	Short: "URL checker on Text files",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("%s \n", red("Missing DOC file"))
			os.Exit(-1)
		}
		if (Debug) {
			log.SetLevel(log.DebugLevel)
		}
		path := args[0]
		GrabUrls(path)
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Debug mode")
}

func GrabUrls(filePath string) {
	start := time.Now()

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Could not open %s \n", red(filePath))
	}

	re := regexp.MustCompile(`(?i)\b((?:https?://|www\d{0,3}[.]|[a-z0-9.\-]+[.][a-z]{2,4}/)(?:[^\s()<>]+|\(([^\s()<>]+|(\([^\s()<>]+\)))*\))+(?:\(([^\s()<>]+|(\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))`)
	urls := re.FindAll(file, -1)
	responses := GetStatusCodes(urls)
	//goodStatus := regexp.MustCompile(`20[0-9]`)
	//errorStatus := regexp.MustCompile(`([4|5][\d]{2}`)
	errors := 0
	ok := 0
	for _, result := range responses {
		if result != nil && result.response != nil {
			ok++;
			fmt.Printf("[%s] %s \n", green(result.response.StatusCode), result.url)
		} else {
			errors++;
			fmt.Printf("[%s] %s \n", red("ERROR"), result.url)
		}
	}
	fmt.Printf("%s URLs checked \n", green(len(responses)))
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s \n", green(elapsed))
	fmt.Printf("Total errors: %s \n", red(errors))
	fmt.Printf("Total OK: %s \n", green(ok))
}

func GetStatusCodes(urls [][]byte) []*HttpResponse {
	ch := make(chan *HttpResponse)
	responses := []*HttpResponse{}
	client := http.Client{
		Timeout: time.Duration(2 * time.Second),
	}
	for _, url := range urls {
		url := string(url)
		go func(url string) {
			log.Debugf("Fetching %s \n", url)
			res, err := client.Get(url)
			ch <- &HttpResponse{url, res, err}
			if err != nil && res != nil && res.StatusCode == http.StatusOK {
				res.Body.Close()
			}
		}(url)
	}

	for {
		select {
		case r := <-ch:
			log.Debugf("%s was fetched ", r.url)
			if r.err != nil {
				log.Debugf("with an error \n", r.err)
			}
			responses = append(responses, r)
			if len(responses) == len(urls) {
				return responses
			}
		}
	}
	return responses
}
