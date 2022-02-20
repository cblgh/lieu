# Lieu

_an alternative search engine_

Created in response to the environs of apathy concerning the use of hypertext
search and discovery. In Lieu, the internet is not what is made searchable, but
instead one's own neighbourhood. Put differently, Lieu is a neighbourhood search
engine, a way for personal webrings to increase serendipitous connexions.

![lieu screenshot](https://user-images.githubusercontent.com/3862362/107115659-75624d80-686e-11eb-81c8-0c6bdec07082.png)


## Goals

* Enable serendipitous discovery
* Support personal communities
* Be reusable, easily

## Usage

```
$ lieu help
Lieu: neighbourhood search engine

Commands
- precrawl  (scrapes config's general.url for a list of links: <li> elements containing an anchor <a> tag)
- crawl     (start crawler, crawls all urls in config's crawler.webring file)
- ingest    (ingest crawled data, generates database)
- search    (interactive cli for searching the database)
- host      (hosts search engine over http)

Example:
    lieu precrawl > data/webring.txt
    lieu crawl > data/crawled.txt
    lieu ingest
    lieu host
```

Lieu's crawl & precrawl commands output to [standard
output](https://en.wikipedia.org/wiki/Standard_streams#Standard_output_(stdout)),
for easy inspection of the data. You typically want to redirect their output to
the files Lieu reads from, as defined in the config file. See below for a
typical workflow.


### Workflow

* Edit the config
* Add domains to crawl in `config.crawler.webring`
	* **If you have a webpage with links you want to crawl:**
	* Set the config's `url` field to that page
	* Populate the list of domains to crawl with `precrawl`: `lieu precrawl > data/webring.txt`
* Crawl: `lieu crawl > data/crawled.txt`
* Create database: `lieu ingest`
* Host engine: `lieu host`

After ingesting the data with `lieu ingest`, you can also use lieu to search the
corpus in the terminal with `lieu search`.

## Theming

Tweak the `theme` values of the config, specified below.

## Config

The config file is written in [TOML](https://toml.io/en/).

```toml
[general]
name = "Merveilles Webring"
# used by the precrawl command and linked to in /about route
url = "https://webring.xxiivv.com"
port = 10001

[theme]
# colors specified in hex (or valid css names) which determine the theme of the lieu instance
foreground = "#ffffff"
background = "#000000"
links = "#ffffff"

[data]
# the source file should contain the crawl command's output 
source = "data/crawled.txt"
# location & name of the sqlite database
database = "data/searchengine.db"
# contains words and phrases disqualifying scraped paragraphs from being presented in search results
heuristics = "data/heuristics.txt"
# aka stopwords, in the search engine biz: https://en.wikipedia.org/wiki/Stop_word
wordlist = "data/wordlist.txt"

[crawler]
# manually curated list of domains, or the output of the precrawl command
webring = "data/webring.txt"
# domains that are banned from being crawled but might originally be part of the webring
bannedDomains = "data/banned-domains.txt"
# file suffixes that are banned from being crawled
bannedSuffixes = "data/banned-suffixes.txt"
# phrases and words which won't be scraped (e.g. if a contained in a link)
boringWords = "data/boring-words.txt"
# domains that won't be output as outgoing links
boringDomains = "data/boring-domains.txt"
```

For your own use, the following config fields should be customized:

* `name`
* `url `
* `port`
* `source`
* `webring`
* `bannedDomains`

The following config-defined files can stay as-is unless you have specific requirements:

* `database`
* `heuristics`
* `wordlist`
* `bannedSuffixes`

For a full rundown of the files and their various jobs, see the [files
description](docs/files.md).

## Developing
Build a binary:
```sh
# this project has an experimental fulltext-search feature, so we need to include sqlite's fts engine (fts5)
go build --tags fts5
# or using go run
go run --tags fts5 . 
```

Create new release binaries:
```sh
./release.sh
```

### License

Source code `AGPL-3.0-or-later`, Inter is available under `SIL OPEN FONT
LICENSE Version 1.1`, Noto Serif is licensed as `Apache License, Version 2.0`.
