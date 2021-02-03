package util

import (
    "os"
	"bytes"
	"encoding/json"
	"fmt"
    "net"
	"io/ioutil"
	"log"
	"strings"

	"lieu/types"
	"github.com/jinzhu/inflection"
	"github.com/komkom/toml"
)

func Inflect(words []string) []string {
	var inflected []string
	for _, word := range words {
		inflected = append(inflected, inflection.Singular(word))
	}
	return inflected
}

func Check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func DatabaseDoesNotExist(filepath string) {
    fmt.Printf("lieu: database %s does not exist\n", filepath)
    fmt.Println("lieu: try running `lieu ingest` if you have already crawled source data")
    Exit()
}

func CheckFileExists (path string) bool {
    _, err := os.Stat(path)
    if err == nil {
        return true
    }
    return os.IsExist(err)
}

func Humanize(n int) string {
	if n > 1000 {
		return fmt.Sprintf("%dk", n/1000)
	} else if n > 1000000 {
		return fmt.Sprintf("%dm", n/1000000)
	}

	return string(n)
}

func Contains(arr []string, query string) bool {
	for _, item := range arr {
		if strings.Contains(query, item) {
			return true
		}
	}
	return false
}

func ReadList(filepath, sep string) []string {
	data, err := ioutil.ReadFile(filepath)
	if err != nil || len(data) == 0{
		return []string{}
	}
	return strings.Split(strings.TrimSuffix(string(data), sep), sep)
}

func CheckPortOpen(port int) bool {
    tcpaddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
    if err != nil {
        return false
    }

    l, err := net.ListenTCP("tcp", tcpaddr)
    defer l.Close()

    if err != nil {
        return false
    }
    return true
}

func ReadConfig() types.Config {
	data, err := ioutil.ReadFile("lieu.toml")
    Check(err)

	var conf types.Config
	decoder := json.NewDecoder(toml.New(bytes.NewBuffer(data)))

	err = decoder.Decode(&conf)
	Check(err)

	return conf
}

func WriteMockConfig () {
    conf := []byte(`[general]
name = "Sweet Webring"
# used by the precrawl command and linked to in /about route
url = "https://example.com/"
port = 10001

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
`)
    err := ioutil.WriteFile("lieu.toml", conf, 0644)
	Check(err)
}

func Exit () {
    os.Exit(0)
}
