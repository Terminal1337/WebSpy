	package main

	import (
		"crypto/tls"
		"fmt"
		"io/ioutil"
		"net/http"
		"os"
		"strconv"
		"strings"
		"sync"
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
		path    string
		wg      sync.WaitGroup
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
		pflag.StringVar(&path, "path", "", "Web Directory or Path (optional)")
		pflag.Parse()

		if pflag.NFlag() < 5 {
			pflag.Usage()
			os.Exit(1)
		}

		ips, err = GoCycle.NewFromFile(fmt.Sprintf("input/%s", file))
		if err != nil {
			logging.Logger.Error().
				Err(err).
				Msg("Readfile")
			os.Exit(1)
		}

		client = &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	func check(ip string) {
		defer wg.Done()

		reqURL := fmt.Sprintf("http://%s/"+path, ip)
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			logging.Logger.Error().
				Err(err).
				Msg("Request Object")
			return
		}

		req.Host = host

		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:125.0) Gecko/20100101 Firefox/125.0")

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
			logging.Logger.Error().
				Err(err).
				Msg("Failed to read response body")
			return
		}

		if strings.Contains(string(b), regex) {
			logging.Logger.Error().
				Str("hit", "true").
				Str("ip", ip).
				Str("status", strconv.Itoa(resp.StatusCode)).
				Msg("Found")
			if err := appendToFile("output/hits.txt", ip+"\n"); err != nil {
				logging.Logger.Error().
					Err(err).
					Msg("Failed to append to file")
			}
		}
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
			wg.Add(1)
			go check(ips.Next())
			checked++
		}
		wg.Wait()
	}
