// Copyright © 2016 Ellison Leão <ellisonleao[at]gmail.com>

package cmd

import (
	"fmt"
	"os"
	"io/ioutil"
	"regexp"
	"net/http"
	"net/url"
	"time"
	"strconv"

	"github.com/spf13/cobra"
	log "github.com/Sirupsen/logrus"
	"github.com/fatih/color"
)

var Debug bool
var Doc string
var Timeout int

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
	RootCmd.PersistentFlags().IntVarP(&Timeout, "timeout", "t", 5,
		"HTTP Request Timeout (seconds)")
}

func GrabUrls(filePath string) {
	start := time.Now()

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Could not open %s \n", red(filePath))
	}

	re := regexp.MustCompile(`(?i)\b((?:https?://|www\d{0,3}[.]|[a-z0-9.\-]+[.][a-z]{2,4}/)(?:[^\s()<>]+|\(([^\s()<>]+|(\([^\s()<>]+\)))*\))+(?:\(([^\s()<>]+|(\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))`)
	urls := re.FindAll(file, -1)
	if urls == nil {
		fmt.Println(red("No URLs could be parsed"))
		os.Exit(-1)
	}
	responses := GetStatusCodes(urls)
	var errors []*HttpResponse
	ok := 0
	var goodStatus bool
	for _, result := range responses {
		if result != nil && result.response != nil {
			ok++;
			status := strconv.Itoa(result.response.StatusCode)
			goodStatus, err = regexp.MatchString("2\\d{2}", status)
			if goodStatus {
				fmt.Printf("[%s] %s \n", green(status), result.url)
			} else {
				fmt.Printf("[%s] %s \n", red(result.response.StatusCode), result.url)
			}
		} else {
			errors = append(errors, result);
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

func GetStatusCodes(urls [][]byte) []*HttpResponse {
	queue := make(chan *HttpResponse)
	responses := []*HttpResponse{}
	client := http.Client{
		Timeout: time.Duration(time.Duration(Timeout) * time.Second),
	}

	for _, urlBytes := range urls {
		u , err := url.Parse(string(urlBytes))
		if err != nil {
			log.Fatalf("Could not parse %s \n", string(urlBytes))
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		go func(url string) {
			log.Debugf("Fetching %s \n", url)
			res, err := client.Head(url)
			queue <- &HttpResponse{url, res, err}
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
