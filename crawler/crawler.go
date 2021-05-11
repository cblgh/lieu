package crawler

import (
	"fmt"
	"lieu/types"
	"lieu/util"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
)

// the following domains are excluded from crawling & indexing, typically because they have a lot of microblog pages
// (very spammy)
func getBannedDomains(path string) []string {
	return util.ReadList(path, "\n")
}

func getBannedSuffixes(path string) []string {
	return util.ReadList(path, "\n")
}

func getBoringWords(path string) []string {
	return util.ReadList(path, "\n")
}

func getBoringDomains(path string) []string {
	return util.ReadList(path, "\n")
}

func find(list []string, query string) bool {
	for _, item := range list {
		if item == query {
			return true
		}
	}
	return false
}

func getLink(target string) string {
	// remove anchor links
	if strings.Contains(target, "#") {
		target = strings.Split(target, "#")[0]
	}
	if strings.Contains(target, "?") {
		target = strings.Split(target, "?")[0]
	}
	target = strings.TrimSpace(target)
	target = strings.ToLower(target)
	// remove trailing /
	return strings.TrimSuffix(target, "/")
}

func getWebringLinks(path string) []string {
	var links []string
	candidates := util.ReadList(path, "\n")
	for _, l := range candidates {
		u, err := url.Parse(l)
		if err != nil {
			continue
		}
		if u.Scheme == "" {
			u.Scheme = "https"
		}
		links = append(links, u.String())
	}
	return links
}

func getDomains(links []string) []string {
	var domains []string
	for _, l := range links {
		u, err := url.Parse(l)
		if err != nil {
			continue
		}
		domains = append(domains, u.Hostname())
	}
	return domains
}

func findSuffix(suffixes []string, query string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToLower(query), suffix) {
			return true
		}
	}
	return false
}

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", " ")
	whitespace := regexp.MustCompile(`\p{Z}`)
	s = whitespace.ReplaceAllString(s, " ")
	return s
}

func handleIndexing(c *colly.Collector) {
	c.OnHTML("meta[name=\"keywords\"]", func(e *colly.HTMLElement) {
		fmt.Println("keywords", cleanText(e.Attr("content")), e.Request.URL)
	})

	c.OnHTML("meta[name=\"description\"]", func(e *colly.HTMLElement) {
		desc := cleanText(e.Attr("content"))
		if len(desc) > 0 {
			fmt.Println("desc", desc, e.Request.URL)
		}
	})

	c.OnHTML("html[lang]", func(e *colly.HTMLElement) {
		lang := cleanText(e.Attr("lang"))
		if len(lang) > 0 {
			fmt.Println("lang", lang, e.Request.URL)
		}
	})

	// get page title
	c.OnHTML("title", func(e *colly.HTMLElement) {
		fmt.Println("title", cleanText(e.Text), e.Request.URL)
	})

	c.OnHTML("body", func(e *colly.HTMLElement) {
		paragraph := cleanText(e.DOM.Find("p").First().Text())
		if len(paragraph) < 1500 && len(paragraph) > 0 {
			fmt.Println("para", paragraph, e.Request.URL)
		}
		// get all relevant page headings
		collectHeadingText("h1", e)
		collectHeadingText("h2", e)
		collectHeadingText("h3", e)
	})
}

func collectHeadingText(heading string, e *colly.HTMLElement) {
	for _, headingText := range e.ChildTexts(heading) {
		if len(headingText) < 500 {
			fmt.Println(heading, cleanText(headingText), e.Request.URL)
		}
	}
}

func Precrawl(config types.Config) {
	res, err := http.Get(config.General.URL)
	util.Check(err)
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatal("status not 200")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	util.Check(err)

	items := make([]string, 0)
	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		if domain, exists := s.Find("a").Attr("href"); exists {
			items = append(items, domain)
		}
	})

	BANNED := getBannedDomains(config.Crawler.BannedDomains)
	for _, item := range items {
		link := getLink(item)
		u, err := url.Parse(link)
		// invalid link
		if err != nil {
			continue
		}
		domain := u.Hostname()
		if find(BANNED, domain) {
			continue
		}
		fmt.Println(link)
	}
}

func Crawl(config types.Config) {
	SUFFIXES := getBannedSuffixes(config.Crawler.BannedSuffixes)
	links := getWebringLinks(config.Crawler.Webring)
	domains := getDomains(links)
	initialDomain := config.General.URL

	// TODO: introduce c2 for scraping links (with depth 1) linked to from webring domains
	// instantiate default collector
	c := colly.NewCollector(
		colly.MaxDepth(3),
	)

	q, _ := queue.New(
		5, /* threads */
		&queue.InMemoryQueueStorage{MaxSize: 100000},
	)

	for _, link := range links {
		q.AddURL(link)
	}

	c.AllowedDomains = domains
	c.AllowURLRevisit = false
	c.DisallowedDomains = getBannedDomains(config.Crawler.BannedDomains)

	delay, _ := time.ParseDuration("200ms")
	c.Limit(&colly.LimitRule{DomainGlob: "*", Delay: delay, Parallelism: 3})

	boringDomains := getBoringDomains(config.Crawler.BoringDomains)
	boringWords := getBoringWords(config.Crawler.BoringWords)

	// on every a element which has an href attribute, call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := getLink(e.Attr("href"))
		if findSuffix(SUFFIXES, link) {
			return
		}
		link = e.Request.AbsoluteURL(link)
		u, err := url.Parse(link)
		// log which site links to what
		if err == nil && !util.Contains(boringWords, link) && !util.Contains(boringDomains, link) {
			outgoingDomain := u.Hostname()
			currentDomain := e.Request.URL.Hostname()
			if !find(domains, outgoingDomain) {
				fmt.Println("non-webring-link", link, e.Request.URL)
				// solidarity! someone in the webring linked to someone else in it
			} else if outgoingDomain != currentDomain && outgoingDomain != initialDomain && currentDomain != initialDomain {
				fmt.Println("webring-link", link, e.Request.URL)
			}
		}
		// only visits links from AllowedDomains
		q.AddURL(link)
	})

	handleIndexing(c)

	// start scraping
	q.Run(c)
}
