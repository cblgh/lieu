package ingest

import (
	"bufio"
	"database/sql"
	"fmt"
	"gomod.cblgh.org/lieu/database"
	"gomod.cblgh.org/lieu/types"
	"gomod.cblgh.org/lieu/util"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/inflection"
)

func partitionSentence(s string) []string {
	punctuation := regexp.MustCompile(`\p{P}`)
	whitespace := regexp.MustCompile(`\p{Z}`)
	invisible := regexp.MustCompile(`\p{C}`)
	symbols := regexp.MustCompile(`\p{S}`)

	s = punctuation.ReplaceAllString(s, " ")
	s = whitespace.ReplaceAllString(s, " ")
	s = invisible.ReplaceAllString(s, " ")
	s = symbols.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "|", " ")
	s = strings.ReplaceAll(s, "/", " ")
	return strings.Fields(s)
}

func filterCommonWords(words, wordlist []string) []string {
	var filtered []string
	for _, word := range words {
		// ingested word was too common, skip it
		if len(word) == 1 || find(wordlist, word) {
			continue
		}
		filtered = append(filtered, inflection.Singular(word))
	}
	return filtered
}

func find(slice []string, sought string) bool {
	for _, item := range slice {
		if item == sought {
			return true
		}
	}
	return false
}

func performAboutHeuristic(heuristicPath, phrase string) bool {
	disallowed := util.ReadList(heuristicPath, "\n")
	ok := !util.Contains(disallowed, phrase)
	return ok && len(phrase) > 20
}

func Ingest(config types.Config) {
	if _, err := os.Stat(config.Data.Database); err == nil || os.IsExist(err) {
		err = os.Remove(config.Data.Database)
		util.Check(err)
	}

	db := database.InitDB(config.Data.Database)
	date := time.Now().Format("2006-01-02")
	database.UpdateCrawlDate(db, date)

	wordlist := util.ReadList(config.Data.Wordlist, "|")

	buf, err := os.Open(config.Data.Source)
	util.Check(err)

	defer func() {
		err = buf.Close()
		util.Check(err)
	}()

	pages := make(map[string]types.PageData)
	var count int
	var batchsize = 100
	batch := make([]types.SearchFragment, 0, 0)
	var externalLinks []string
	paragraphPairs := make([]types.WholeParagraph, 0, 0)

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()
		firstSpace := strings.Index(line, " ")
		lastSpace := strings.LastIndex(line, " ")

		if len(line) == 0 || firstSpace == -1 {
			continue
		}

		pageurl := strings.TrimSuffix(strings.TrimSpace(line[lastSpace:len(line)]), "/")
		if !strings.HasPrefix(pageurl, "http") {
			continue
		}

		var page types.PageData
		if data, exists := pages[pageurl]; exists {
			page = data
		} else {
			page.URL = pageurl
		}

		token := line[0:firstSpace]
		rawdata := strings.TrimSpace(line[firstSpace:lastSpace])
		payload := strings.ToLower(rawdata)

		var processed []string
		score := 1
		switch token {
		case "title":
			if len(page.About) == 0 {
				page.About = rawdata
				page.AboutSource = token
			}
			score = 5
			page.Title = rawdata
			processed = partitionSentence(payload)
		case "h1":
			if len(page.About) == 0 {
				page.About = rawdata
				page.AboutSource = token
			}
			fallthrough
		case "h2":
			fallthrough
		case "h3":
			score = 15
			processed = partitionSentence(payload)
		case "desc":
			if len(page.About) < 30 && len(rawdata) < 100 && len(rawdata) > len(page.About) {
				page.About = rawdata
				page.AboutSource = token
			}
			processed = partitionSentence(payload)
		case "og-desc":
			page.About = rawdata
			page.AboutSource = token
			processed = partitionSentence(payload)
		case "para":
			if page.AboutSource != "og-desc" || len(rawdata)*10 > len(page.About)*7 {
				if performAboutHeuristic(config.Data.Heuristics, payload) {
					page.About = rawdata
					page.AboutSource = token
				}
			}
			processed = partitionSentence(payload)
		case "lang":
			page.Lang = rawdata
		case "keywords":
			processed = strings.Split(strings.ReplaceAll(payload, ", ", ","), ",")
		case "non-webring-link":
			externalLinks = append(externalLinks, rawdata)
		case "big-para":
			paragraphPairs = append(paragraphPairs, types.WholeParagraph{Text: rawdata, URL: pageurl})
		default:
			continue
		}

		pages[pageurl] = page
		processed = filterCommonWords(processed, wordlist)
		count += len(processed)

		for _, word := range processed {
			batch = append(batch, types.SearchFragment{Word: word, URL: pageurl, Score: score})
		}
		if token == "title" {
			// only extract path segments once per url.
			// we do it here because every page is virtually guaranteed to have a title attr &
			// it only appears once
			for _, word := range extractPathSegments(strings.ToLower(pageurl)) {
				batch = append(batch, types.SearchFragment{Word: word, URL: pageurl, Score: 2})
			}
		}

		if len(pages) > batchsize {
			ingestBatch(db, batch, pages, externalLinks, paragraphPairs)
			externalLinks = make([]string, 0, 0)
			paragraphPairs = make([]types.WholeParagraph, 0, 0)
			batch = make([]types.SearchFragment, 0, 0)
			// TODO: make sure we don't partially insert any page data
			pages = make(map[string]types.PageData)
		}
	}
	ingestBatch(db, batch, pages, externalLinks, paragraphPairs)
	fmt.Printf("ingested %d words\n", count)

	err = scanner.Err()
	util.Check(err)
}

func ingestBatch(db *sql.DB, batch []types.SearchFragment, pageMap map[string]types.PageData, links []string, paragraphPairs []types.WholeParagraph) {
	pages := make([]types.PageData, len(pageMap))
	i := 0
	for k := range pageMap {
		pages[i] = pageMap[k]
		i++
	}
	// TODO (2021-11-10): debug the "incomplete input" error / log, and find out where it is coming from
	log.Println("starting to ingest batch (Pages:", len(pages), "Words:", len(batch), "Links:", len(links), ")")
	database.InsertManyDomains(db, pages)
	database.InsertManyPages(db, pages)
	for i := 0; i < len(batch); i += 3000 {
		end_i := i + 3000
		if end_i > len(batch) {
			end_i = len(batch)
		}
		database.InsertManyWords(db, batch[i:end_i])
	}
	database.InsertManyExternalLinks(db, links)
	database.InsertManyBigParagraphs(db, paragraphPairs)
	log.Println("finished ingesting batch")
}

func extractPathSegments(pageurl string) []string {
	u, err := url.Parse(pageurl)
	util.Check(err)
	if len(u.Path) == 0 {
		return make([]string, 0, 0)
	}
	s := u.Path
	s = strings.TrimSuffix(s, ".html")
	s = strings.TrimSuffix(s, ".htm")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ToLower(s)
	return strings.Fields(s)
}
