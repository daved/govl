// Copyright © 2016 Ellison Leão <ellisonleao[at]gmail.com>

package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var debug bool
var doc string
var timeout int

var yellow = color.New(color.FgYellow).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()

type httpResponse struct {
	url      string
	response *http.Response
	err      error
}

var vl = &cobra.Command{
	Use:   "vl",
	Short: "URL checker on Text files",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("%s \n", red("Missing DOC file"))
			os.Exit(-1)
		}
		if debug {
			log.SetLevel(log.DebugLevel)
		}
		path := args[0]
		grabUrls(path)
	},
}

// Execute is wrapper for the vl command
func Execute() {
	if err := vl.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	vl.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Debug mode")
	vl.PersistentFlags().IntVarP(&timeout, "timeout", "t", 1,
		"HTTP Request Timeout (seconds)")
}

func grabUrls(filePath string) {
	start := time.Now()

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Could not open %s \n", red(filePath))
		os.Exit(-1)
	}

	re := regexp.MustCompile(`(?i)\b((?:https?://|www\d{0,3}[.]|[a-z0-9.\-]+[.][a-z]{2,4}/)(?:[^\s()<>]+|\(([^\s()<>]+|(\([^\s()<>]+\)))*\))+(?:\(([^\s()<>]+|(\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))`)
	urls := re.FindAll(file, -1)
	if urls == nil {
		fmt.Println(red("No URLs could be parsed"))
		os.Exit(-1)
	}
	responses := getStatusCodes(urls)
	var errors []*httpResponse
	ok := 0
	var goodStatus bool
	for _, result := range responses {
		if result != nil && result.response != nil {
			ok++
			status := strconv.Itoa(result.response.StatusCode)
			goodStatus, err = regexp.MatchString("2\\d{2}", status)
			if goodStatus {
				fmt.Printf("[%s] %s \n", green(status), result.url)
			} else {
				fmt.Printf("[%s] %s \n", red(result.response.StatusCode), result.url)
			}
		} else {
			errors = append(errors, result)
			fmt.Printf("[%s] %s \n", red("ERROR"), result.url)
		}
	}
	fmt.Printf("%s URLs checked \n", green(len(responses)))
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s \n", green(elapsed))
	fmt.Printf("Total errors: %s \n", red(len(errors)))
	fmt.Printf("Total OK: %s \n", green(ok))

	if len(errors) > 0 {
		fmt.Println(red("Some errors were found (Please check for false positives):"))
		for _, result := range errors {
			fmt.Printf("- [%s] Error: \n %s \n", yellow(result.url), red(result.err))
		}
		os.Exit(-1)
	}
}

func getStatusCodes(urls [][]byte) []*httpResponse {
	queue := make(chan *httpResponse)
	responses := []*httpResponse{}
	client := http.Client{
		Timeout: time.Duration(time.Duration(timeout) * time.Second),
	}

	for _, urlBytes := range urls {
		u, err := url.Parse(string(urlBytes))
		if err != nil {
			log.Fatalf("Could not parse %s \n", string(urlBytes))
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		go func(url string) {
			log.Debugf("Fetching %s \n", url)
			res, err := client.Head(url)
			queue <- &httpResponse{url, res, err}
			if err != nil && res != nil && res.StatusCode == http.StatusOK {
				res.Body.Close()
			}
		}(u.String())
	}

	for r := range queue {
		log.Debugf("%s was fetched ", r.url)
		if r.err != nil {
			log.Debugf("with an error \n", r.err)
		}
		responses = append(responses, r)
		if len(responses) == len(urls) {
			return responses
		}
	}
	return responses
}
