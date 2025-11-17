package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"

	"gomod.cblgh.org/lieu/types"

	"github.com/microcosm-cc/bluemonday"
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

// document.querySelectorAll-type functionality. limited functionality as of now (no classes or id support atm, i think!!)
func QuerySelector(query string, current *goquery.Selection, results *[]string) {
	var op, operand string

	attrPattern := regexp.MustCompile(`(\w+)\[(\w+)\](.+)?`)
	attrValuePattern := regexp.MustCompile(`\[(\w+)\]`)

	if len(query) == 0 {
		return
	}

	fields := strings.Fields(query)
	part := fields[0]
	query = strings.Join(fields[1:], " ")
	if part == ">" {
		op = "subchild"
	} else if attrPattern.MatchString(part) {
		op = "element"
		matches := attrPattern.FindStringSubmatch(part)
		operand = matches[1]
		var optional string
		if len(matches) == 4 {
			optional = matches[3]
		}
		query = strings.TrimSpace(fmt.Sprintf("[%s]%s %s", matches[2], optional, query))
	} else if attrValuePattern.MatchString(part) {
		op = "attr"
		operand = attrValuePattern.FindStringSubmatch(part)[1]
	} else if len(query) == 0 {
		op = "final"
	} else {
		op = "element"
		operand = part
	}

	switch op {
	case "element": // e.g. [el]; bla > [el]; but also [el] > bla
		current = current.Find(operand)
		if strings.HasSuffix(query, "first-of-type") {
			break
		}
		fallthrough
	case "subchild": // [preceding] > [future]
		// recurse querySelector on all [preceding] element types
		current.Each(func(j int, s *goquery.Selection) {
			QuerySelector(query, s, results)
		})
		return
	case "attr": // x[attr]
		// extract the attribute
		if str, exists := current.Attr(operand); exists {
			*results = append(*results, str)
		}
		return
	case "final": // no more in query, and we did not end on an attr: get text
		*results = append(*results, current.Text())
	}
	QuerySelector(query, current, results)
}

func DatabaseDoesNotExist(filepath string) {
	fmt.Printf("lieu: database %s does not exist\n", filepath)
	fmt.Println("lieu: try running `lieu ingest` if you have already crawled source data")
	Exit()
}

func CheckFileExists(path string) bool {
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

	return fmt.Sprintf("%d", n)
}

var contentPolicy = bluemonday.StrictPolicy() // remove all html tags and possible XSS from the input
var whitespacePattern = regexp.MustCompile(`\p{Z}+`)

func CleanText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = whitespacePattern.ReplaceAllString(s, " ")
	return s
}

func CleanTextStrict(s string) string {
	return contentPolicy.Sanitize(CleanText(s))
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
	if err != nil || len(data) == 0 {
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

func WriteMockConfig() {
	conf := []byte(`[general]
name = "Sweet Webring"
# used by the precrawl command and linked to in /about route
url = "https://example.com/"
webringSelector = "li > a"
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
# queries to search for finding preview text
previewQueryList = "data/preview-query-list.txt"
`)
	err := ioutil.WriteFile("lieu.toml", conf, 0644)
	Check(err)
}

func Exit() {
	os.Exit(0)
}

func DeduplicateSlice(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
