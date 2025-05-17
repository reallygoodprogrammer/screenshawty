# screenshawty

A quick go application for taking a screenshot of a webpage and writing
a list of words that appear within the rendered text using playwright-go. 
I use this for scraping visible page data in other projects quickly.

## install

```
# install playwright-go
go install github.com/playwright-community/playwright-go/cmd/playwright@v0.xxxx.x
playwright install --with-deps

# install screenshawty
go install github.com/reallygoodprogrammer/screenshawty@latest
```

## example and usage

```
# pipe urls into screenshawty
# writes files to ./shawty_output/...
cat urls | screenshawty
```

```
Usage of screenshawty:
  -concurrency int
        concurrency for requests (default 5)
  -dir string
        directory to write data to (default "shawty_output")
  -help
        display help message
  -timeout float
        timeout in ms (default 10000)
```
