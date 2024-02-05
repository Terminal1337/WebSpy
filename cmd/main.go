package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"webspy/pkg/logging"

	"github.com/Terminal1337/GoCycle"
	"github.com/ogier/pflag"
	"github.com/zenthangplus/goccm"
)

var (
	ips     *GoCycle.Cycle
	threads int
	file    string
	regex   string
	timeout int
	client  *http.Client
	host    string
	checked int
)

func appendToFile(filename string, text string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(text); err != nil {
		return err
	}

	return nil
}

func init() {
	var err error

	pflag.StringVar(&file, "file", "80.txt", "ips list file")
	pflag.StringVar(&regex, "regex", "<html>", "html regex to compare")
	pflag.StringVar(&host, "host", "example.com", "domain or hostname")
	pflag.IntVar(&threads, "threads", 1000, "number of threads")
	pflag.IntVar(&timeout, "timeout", 5, "http timeout in seconds")
	pflag.Parse()
	// if pflag.NFlag() == 0 {
	// 	pflag.Usage()
	// 	os.Exit(0)
	// }
	ips, err = GoCycle.NewFromFile(fmt.Sprintf("input/%s", file))
	if err != nil {
		logging.Logger.Error().
			Err(err).
			Msg("Readfile")
		os.Exit(0)
	}
	client = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

}

func check(ip string) {
	reqURL := fmt.Sprintf("http://%s/login", ip)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		// logging.Logger.Error().
		// 	Err(err).
		// 	Msg("Request Object")
		return
	}

	
	req.Host = host

	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		// logging.Logger.Error().
		// 	Err(err).
		// 	Msg("Request Object")
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// logging.Logger.Error().
		// 	Err(err).
		// 	Msg("Failed to read response body")
		resp.Body.Close()
		return
	}

	if strings.Contains(string(b), regex) {
		logging.Logger.Error().
			Str("hit", "true").
			Str("ip", ip).
			Str("status", strconv.Itoa(resp.StatusCode)).
			Msg("Found")
		appendToFile("output/hits.txt", ip+"\n")

	}
	return
}
func status() {
	for {
		logging.Logger.Error().
			Str("total", strconv.Itoa(ips.ListLength())).
			Str("checked", strconv.Itoa(checked)).
			Str("remaining", strconv.Itoa(ips.ListLength()-checked)).
			Str("threads", strconv.Itoa(threads)).
			Str("host", host).
			Str("regex", regex).
			Msg("status")
		time.Sleep(5 * time.Second)
	}
}
func main() {
	c := goccm.New(threads)
	go status()
	for i := 1; i <= ips.ListLength(); i++ {
		c.Wait()
		go func() {
			check(ips.Next())
			checked += 1
			c.Done()

		}()
	}
	c.WaitAllDone()
}
