package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type PageContent struct {
	Code  int
	Title string
}

var numFetchers int

var dataMutex = sync.Mutex{}
var data = map[string]PageContent{}
var toFetch chan string
var aborting chan int
var threadsShutdown = sync.WaitGroup{}

func init() {
	flag.IntVar(&numFetchers, "concurrency", 16, "Number of concurrent fetcher threads")
}

type ParsedPage struct {
	Title string
	Links []string
}

func parsePage(baseUrl string, body string) ParsedPage {
	result := ParsedPage{}
	tkn := html.NewTokenizer(strings.NewReader(body))
	var isTitle bool
	for {
		tt := tkn.Next()
		if tt == html.ErrorToken {
			break
		}
		switch {
		case tt == html.StartTagToken:
			t := tkn.Token()
			isTitle = t.Data == "title"
			if t.Data == "a" {
				for _, a := range t.Attr {
					if a.Key == "href" {
						if !strings.HasPrefix(a.Val, "http://") && !strings.HasPrefix(a.Val, "https://") {
							result.Links = append(result.Links, baseUrl+a.Val)
						} else {
							result.Links = append(result.Links, a.Val)
						}
						break
					}
				}
			}
		case tt == html.TextToken:
			t := tkn.Token()
			if isTitle {
				result.Title = t.Data
			}
			isTitle = false
		}
	}
	return result
}

func ReaderThread() {
	defer threadsShutdown.Done()

TryFetch:
	thisUrl := "<waiting>"
	for thisUrl == "<waiting>" {
		select {
		case msgAbort := <-aborting:
			_ = msgAbort
			return
		case thisUrl = <-toFetch:
		}
	}

	// Check for existing URL
	dataMutex.Lock()
	entry, ok := data[thisUrl]
	if ok {
		dataMutex.Unlock()
		goto TryFetch // This url has already been processed
	}
	entry = PageContent{Code: -1} // Insert a map entry, we'll fill it in shortly
	data[thisUrl] = entry
	dataMutex.Unlock()

	// Fetch the page
	resp, err := http.Get(thisUrl)
	if err != nil {
		e := fmt.Errorf("warning, could not fetch %s: %w", thisUrl, err)
		fmt.Printf("%s\n", e)
		goto TryFetch
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e := fmt.Errorf("warning, could not fetch %s: %w", thisUrl, err)
		fmt.Printf("%s\n", e)
		goto TryFetch
	}

	// Parse the page
	urlParsed, _ := url.Parse(thisUrl)
	baseUrl := urlParsed.Scheme + "://" + urlParsed.Hostname()
	parsed := parsePage(baseUrl, string(body))

	// Record the data and push links into the fetch channel
	entry.Code = resp.StatusCode
	entry.Title = parsed.Title
	dataMutex.Lock()
	data[thisUrl] = entry
	dataMutex.Unlock()
	/*
		for _, link := range parsed.Links {
			select {
			case msgAbort := <-aborting:
				_ = msgAbort
				return
			case toFetch <- link:
			}
		}
	*/
	for _, link := range parsed.Links {
		toFetch <- link
	}

	goto TryFetch
}

func displayList() {
	dataMutex.Lock()
	for thisUrl, entry := range data {
		fmt.Printf("%s (%s: %d)\n", entry.Title, thisUrl, entry.Code)
	}
	dataMutex.Unlock()
}

func displayQuery(q string) {
	qLocase := strings.ToLower(q)
	dataMutex.Lock()
	for thisUrl, entry := range data {
		entryLocase := strings.ToLower(entry.Title)
		if strings.Contains(entryLocase, qLocase) {
			fmt.Printf("found in %s (%s: %d)\n", entry.Title, thisUrl, entry.Code)
		}
	}
	dataMutex.Unlock()
}

func main() {
	flag.Parse()
	StartUrl := flag.Arg(0)
	if StartUrl == "" {
		fmt.Print("Please provide an URL as a command argument")
		return
	}

	toFetch = make(chan string, 10000000) // Max total number of requests
	aborting = make(chan int, numFetchers)
	for i := 0; i < numFetchers; i++ {
		threadsShutdown.Add(1)
		go ReaderThread()
	}
	toFetch <- StartUrl

	// Simple console implementation
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nCommands: l to list, q<query> to query titles, a to abort\n->")
		text, _ := reader.ReadString('\n')
		text = strings.ReplaceAll(text, "\r", "")
		text = strings.TrimSuffix(text, "\n")

		if strings.HasPrefix(text, "a") {
			break
		} else if strings.HasPrefix(text, "l") {
			displayList()
		} else if strings.HasPrefix(text, "q") {
			query := strings.TrimPrefix(text, "q")
			displayQuery(strings.TrimSpace(query))
		} else {
			fmt.Print("Unknown command\n")
		}
	}

	for i := 0; i < numFetchers; i++ {
		aborting <- 1
	}
	threadsShutdown.Wait()
	close(toFetch)
}
