package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	var con int
	var dir string
	var help bool
	var timeout float64
	var waitTime int
	var openBrowser bool
	var wordFile bool
	var headerFile bool
	var bodyFile bool
	var allFile bool

	flag.StringVar(&dir, "dir", "shawty_output", "directory to write data to")
	flag.IntVar(&con, "concurrency", 5, "concurrency for requests (default=5)")
	flag.Float64Var(&timeout, "timeout", 10000, "timeout in ms")
	flag.IntVar(&waitTime, "wait-time", 6, "wait time before taking screenshot (default=6)")
	flag.BoolVar(&openBrowser, "browser", false, "don't run headless (default=false)")
	flag.BoolVar(&wordFile, "words", false, "create word files containing all words from resp body")
	flag.BoolVar(&headerFile, "headers", false, "create files containing the headers of the response")
	flag.BoolVar(&bodyFile, "body", false, "create files containing the body of the response")
	flag.BoolVar(&allFile, "all", false, "create two files containing (1) the body and (2) headers")
	flag.BoolVar(&help, "help", false, "display help message")
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if allFile {
		bodyFile = true
		headerFile = true
	}

	err := os.Mkdir(dir, 0750)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create directory: %v\n", err)
		return
	}
	args := flag.Args()

	pw, err := playwright.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start playwright: %v\n", err)
		return
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(!openBrowser),
		Args:     []string{"--no-sandbox"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start browser: %v\n", err)
		return
	}

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  1280,
			Height: 720,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create context: %v\n", err)
		return
	}

	if openBrowser {
		headers := map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
			"Accept-Language": "en-US,en;q=0.9",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		}
		context.SetExtraHTTPHeaders(headers)
	}

	wait := time.Duration(waitTime) * time.Second

	urls := make(chan string)
	output := make(chan string)

	var inGrp sync.WaitGroup
	for i := 0; i < con; i++ {
		inGrp.Add(1)
		go func() {
			defer inGrp.Done()
			for url := range urls {
				// create new page
				page, err := context.NewPage()
				if err != nil {
					fmt.Fprintf(os.Stderr, "error creating page: %v\n", err)
					continue
				}

				// go to url
				response, err := page.Goto(url, playwright.PageGotoOptions{
					Timeout: playwright.Float(timeout),
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "error going to url: %v\n", err)
				} else {
					time.Sleep(wait)

					urlDir := dir+"/"+strings.ReplaceAll(url, "/", "_")
					err = os.Mkdir(urlDir, 0750)
					if err != nil {
						fmt.Fprintf(os.Stderr, "could not create directory: %v\n", err)
						err = page.Close()
						if err != nil {
							fmt.Fprintf(os.Stderr, "error closing page: %v\n", err)
						}
						continue
					}

					// take screenshot
					_, err = page.Screenshot(playwright.PageScreenshotOptions{
						Path: playwright.String(urlDir + "/screenshot.png"),
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "error taking screenshot: %v\n", err)
					}

					// if writing word file, extract words, write to file
					if wordFile {
						result, err := page.Evaluate("(document.body.innerText)")
						if err != nil {
							fmt.Fprintf(os.Stderr, "could not evaluate js: %v\n", err)
						} else {
							text := result.(string)
							words := strings.Fields(text)
							file, err := os.Create(urlDir + "/words.txt")
							if err != nil {
								fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
							} else {
								defer file.Close()
								for _, w := range words {
									_, err := file.WriteString(w + "\n")
									if err != nil {
										fmt.Fprintf(os.Stderr, "error writing to file: %v\n", err)
										continue
									}
								}
							}
						}
					}
					
					// if writing response headers, extract, write to file
					if headerFile {
						headers := response.Headers();
						file, err := os.Create(urlDir + "/headers.resp")
						if err != nil {
							fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
						} else {
							defer file.Close()
							for k, v := range headers {
								_, err := file.WriteString(k+": "+v+"\n")
								if err != nil {
									fmt.Fprintf(os.Stderr, "error writing to file: %v\n", err)
									continue
								}
							}
						}
					}

					// if writing response body, extract, write to file
					if bodyFile {
						body, err := response.Body()
						if err != nil {
							fmt.Fprintf(os.Stderr, "error getting response body: %v\n", err)
						} else {
							file, err := os.Create(urlDir + "/body.resp")
							if err != nil {
								fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
								continue
							}
							defer file.Close()
							file.WriteString(string(body))
						}
					}
					output <- "[+] "+url
				}
				err = page.Close()
				if err != nil {
					fmt.Fprintf(os.Stderr, "error closing page: %v\n", err)
				}
			}
		}()
	}

	var outGrp sync.WaitGroup
	outGrp.Add(1)
	go func() {
		defer outGrp.Done()
		for output := range output {
			fmt.Printf("%s\n", output)
		}
	}()

	if len(args) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			url := scanner.Text()
			urls <- url
		}
	} else {
		for _, url := range args {
			urls <- url
		}
	}

	close(urls)
	inGrp.Wait()
	browser.Close()
	close(output)
	outGrp.Wait()
}
