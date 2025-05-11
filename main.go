package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"os"
	"strings"
	"sync"
)

func main() {
	var con int
	var dir string
	var help bool
	var timeout float64

	flag.StringVar(&dir, "dir", "shawty_output", "directory to write data to")
	flag.IntVar(&con, "concurrency", 5, "concurrency for requests")
	flag.Float64Var(&timeout, "timeout", 10000, "timeout in ms")
	flag.BoolVar(&help, "help", false, "display help message")
	flag.Parse()

	if help {
		flag.Usage()
		return
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
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start browser: %v\n", err)
		return
	}
	
	urls := make(chan string)
	output := make(chan string)

	var inGrp sync.WaitGroup
	for i := 0; i < con; i++ {
		inGrp.Add(1)
		go func() {
			defer inGrp.Done()
			for url := range urls {
				page, err := browser.NewPage()
				if err != nil {
					fmt.Fprintf(os.Stderr, "error creating page: %v\n", err)
					continue
				}

				_, err = page.Goto(url, playwright.PageGotoOptions{
					Timeout: playwright.Float(timeout),
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "error going to url: %v\n", err)
				} else {
					fileName := strings.ReplaceAll(url, "/", "_")
					_, err = page.Screenshot(playwright.PageScreenshotOptions{
						Path: playwright.String(dir+"/"+fileName+"_screenshot.png"),
						FullPage: playwright.Bool(true),
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "error taking screenshot: %v\n", err)
					} else {
						output <- "[+] screenshot successful: "+fileName+"_screenshot.png"
					}

					result, err := page.Evaluate("(document.body.innerText)")
					if err != nil {
						fmt.Fprintf(os.Stderr, "could not evaluate js: %v\n", err)
					} else {
						text := result.(string)
						words := strings.Fields(text)
						file, err := os.Create(dir+"/"+fileName+"_words.txt")
						if err != nil {
							fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
						} else { 
							for _, w := range words {
								_, err := file.WriteString(w+"\n")
								if err != nil {
									fmt.Fprintf(os.Stderr, "error writing to file : %v\n", err)
									return
								}
							}
							output <- "[+] word file saved: "+fileName+"_words.txt"
						}
					}
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
