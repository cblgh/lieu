package database

/* example query
SELECT p.url
FROM inv_index index
INNER JOIN pages p ON p.id = index.pageid
WHERE i.word = "project";

select url from inv_index where word="esoteric" group by url order by sum(score) desc limit 15;

select url from inv_index where word = "<word>" group by url order by sum(score) desc;
*/

import (
	"database/sql"
	"fmt"
	"lieu/types"
	"lieu/util"
	"html/template"
	"log"
	"net/url"
	"strings"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
)

var languageCodeSanityRegex = regexp.MustCompile("^[a-zA-Z\\-0-9]+$")

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalln(err)
	}
	if db == nil {
		log.Fatalln("db is nil")
	}
	createTables(db)
	return db
}

func createTables(db *sql.DB) {
	// create the table if it doesn't exist
	queries := []string{`
    CREATE TABLE IF NOT EXISTS domains (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        domain TEXT NOT NULL UNIQUE
    );
    `,
		`
    CREATE TABLE IF NOT EXISTS stats (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        last_crawl TEXT
    );
    `,
		`
    CREATE TABLE IF NOT EXISTS pages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT NOT NULL UNIQUE,
        title TEXT,
        about TEXT,
        lang TEXT,
        domain TEXT NOT NULL,
        FOREIGN KEY(domain) REFERENCES domains(domain)
    );
    `,
		`
    CREATE TABLE IF NOT EXISTS external_pages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        url TEXT NOT NULL UNIQUE,
        domain TEXT NOT NULL,
        title TEXT
    );
    `,
		`
    CREATE TABLE IF NOT EXISTS inv_index (
        word TEXT NOT NULL,
        score INTEGER NOT NULL,
        url TEXT NOT NULL,
        FOREIGN KEY(url) REFERENCES pages(url)
    )`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS external_links USING fts5 (url, tokenize="trigram")`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS big_search USING fts5 (text, url, tokenize="porter")`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Fatalln(fmt.Errorf("failed to execute %s (%w)", query, err))
		}
	}
}

/* TODO: filters
lang:en|fr|en|<..>
nosite:excluded-domain.com

"word1 word2 word3" strict query

query params:
&order=score, &order=count
*/

var emptyStringArray = []string{}

func SearchWordsByScore(db *sql.DB, words []string) []types.PageData {
	return SearchWords(db, words, true, emptyStringArray, emptyStringArray, emptyStringArray)
}

func SearchWordsBySite(db *sql.DB, words []string, domain string) []types.PageData {
	// search words by site is same as search words by score, but adds a domain condition
	return SearchWords(db, words, true, []string{domain}, emptyStringArray, emptyStringArray)
}

func SearchWordsByCount(db *sql.DB, words []string) []types.PageData {
	return SearchWords(db, words, false, emptyStringArray, emptyStringArray, emptyStringArray)
}

func FulltextSearchWords(db *sql.DB, phrase string) []types.PageData {
	query := fmt.Sprintf(`SELECT url from external_links WHERE url MATCH ? GROUP BY url ORDER BY RANDOM() LIMIT 30`)

	stmt, err := db.Prepare(query)
	util.Check(err)
	defer stmt.Close()

	rows, err := stmt.Query(phrase)
	util.Check(err)
	defer rows.Close()

	var pageData types.PageData
	var pages []types.PageData
	for rows.Next() {
		if err := rows.Scan(&pageData.URL); err != nil {
			log.Fatalln(err)
		}
		pageData.Title = pageData.URL
		pages = append(pages, pageData)
	}
	return pages
}

func FulltextSearchWholeParagraphs(db *sql.DB, phrase string, domain []string, nodomain []string) []types.PageData {
	var args []interface{}
	args = append(args, phrase)

	domains := []string{"1"}
	if len(domain) > 0 && domain[0] != "" {
		domains = make([]string, 0) // we've got at least one domain! clear domains default
		for _, d := range domain {
			domains = append(domains, "p.domain = ?")
			args = append(args, d)
		}
	}

	nodomains := []string{"1"}
	if len(nodomain) > 0 && nodomain[0] != "" {
		nodomains = make([]string, 0)
		for _, d := range nodomain {
			nodomains = append(nodomains, "p.domain != ?")
			args = append(args, d)
		}
	}

	query := fmt.Sprintf(`
	SELECT bs.text, p.about, HIGHLIGHT(big_search, 0, '<strong>', '</strong>'), bs.url FROM big_search bs INNER JOIN pages p ON bs.url = p.url 
	WHERE bs.text MATCH ? 
  AND (%s)
  AND (%s)
	ORDER BY bs.rank LIMIT 30
	`, strings.Join(domains, " OR "), strings.Join(nodomains, " AND "))

	 // select word, url from inv_index where url = (select distinct url from inv_index limit 100);

	// 1: SELECT HIGHLIGHT(big_search, 0, '<strong>', '</strong>'), url from big_search WHERE text MATCH ? ORDER BY rank LIMIT 30
  //
	// 2: SELECT HIGHLIGHT(big_search, 0, '<strong>', '</strong>'), url from big_search WHERE text MATCH ? GROUP BY url ORDER BY rank LIMIT 30
  //
	// 3: SELECT text, url from big_search WHERE text MATCH ? GROUP BY url ORDER BY rank LIMIT 30

	stmt, err := db.Prepare(query)
	util.Check(err)
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	util.Check(err)
	defer rows.Close()

	var pageData types.PageData
	var pages []types.PageData
	// `duplicates` keeps track of whether the same text has been returned already -> deduplicates search results. 
	//
	// rationale: for the dataset i am testing this on (merveilles forum) the links to the same thread can change, and so
	// it's more useful to track dupes on a per paragraph basis than per-url.
	duplicates := make(map[string]bool)
	var paragraphMatch string
	var unadornedParagraphMatch string
	for rows.Next() {
		if err := rows.Scan(&unadornedParagraphMatch, &pageData.About, &paragraphMatch, &pageData.URL); err != nil {
			log.Fatalln(err)
		}
		if _, exists := duplicates[paragraphMatch]; !exists {
			pageData.About = pageData.About
			// both About and the fts paragraph contain the same, null the about paragraph
			// note: the crawled data represented by `unadornedParagarphMatch` has been run through bluemonday's strict
			// santiization. this means a lot of html escapes are present in the text. in order to do a fair comparison, we
			// need to run the same sanitzation on pageData.About for comparing purposes. 
			//
			// secondary note: pageData.About *does not* need to be sanitize - it is not rendered using the `unsafe` 
			// template.HTML* type; golang's templates will autoescape its content as needed.
			if strings.EqualFold(unadornedParagraphMatch, util.CleanTextStrict(pageData.About)) {
				pageData.About = ""
			}
			// TODO (2024-10-10): surface search syntax legend (details/summary dropdown with pos absolute?) displaying 
			// tricks for links and paragraph search respectively

			// TODO (2024-10-10): convert ParagraphResult to be []template.HTML and collect paragraphs per unique url. 
			// if a len(slice) > 1, render in template as <ul><li>..</li></ul> for all the paragraph matches (but not the pageData.About)
			// example https://i.cblgh.org/2024-10/11788Ohq.png

			// IDEA (2024-10-11): augment "links search" by using the results returned by it, to then perform a FTS search
			// `where url = ? OR url = ? ..." for each url returned by the links search results. 
			//  
			// this offers a greater precision (maybe?) using the initial weighted ranking, and then gets us depth of results
			// by running a constrained fts
			pageData.URL = pageData.URL
			pageData.ParagraphResult = template.HTML(paragraphMatch)
			pages = append(pages, pageData)
			duplicates[paragraphMatch] = true
		}
	}
	return pages
}

func UpdateCrawlDate(db *sql.DB, date string) {
	stmt := `INSERT OR IGNORE INTO stats(last_crawl) VALUES (?)`
	_, err := db.Exec(stmt, date)
	if err != nil {
		util.Check(fmt.Errorf("failed to update crawl date (%w)", err))
	}
}

func GetLastCrawl(db *sql.DB) string {
	rows, err := db.Query("SELECT last_crawl FROM stats WHERE last_crawl IS NOT NULL ORDER BY id DESC LIMIT 1")
	util.Check(err)
	defer rows.Close()

	var date string
	for rows.Next() {
		err = rows.Scan(&date)
		if err != nil {
			util.Check(fmt.Errorf("failed to get last crawl (%w)", err))
		}
	}
	return date
}

func GetDomainCount(db *sql.DB) int {
	return countQuery(db, "domains")
}

func GetPageCount(db *sql.DB) int {
	return countQuery(db, "pages")
}

func GetWordCount(db *sql.DB) int {
	return countQuery(db, "inv_index")
}

func GetRandomDomain(db *sql.DB) string {
	rows, err := db.Query("SELECT domain FROM domains ORDER BY RANDOM() LIMIT 1;")
	util.Check(err)
	defer rows.Close()

	var domain string
	for rows.Next() {
		err = rows.Scan(&domain)
		util.Check(err)
	}
	return domain
}

func GetRandomExternalLink(db *sql.DB) string {
	rows, err := db.Query("SELECT url FROM external_links ORDER BY RANDOM() LIMIT 1;")
	util.Check(err)
	defer rows.Close()

	var link string
	for rows.Next() {
		err = rows.Scan(&link)
		util.Check(err)
	}
	return link
}

func GetRandomPage(db *sql.DB) string {
	domain := GetRandomDomain(db)
	stmt, err := db.Prepare("SELECT url FROM pages WHERE domain = ? ORDER BY RANDOM() LIMIT 1;")
	defer stmt.Close()
	util.Check(err)

	rows, err := stmt.Query(domain)
	defer rows.Close()

	var link string
	for rows.Next() {
		err = rows.Scan(&link)
		util.Check(err)
	}
	return link
}

func countQuery(db *sql.DB, table string) int {
	rows, err := db.Query(fmt.Sprintf("SELECT COUNT(*) FROM %s;", table))
	util.Check(err)
	defer rows.Close()

	var count int
	for rows.Next() {
		err = rows.Scan(&count)
		util.Check(err)
	}
	return count
}

func SearchWords(db *sql.DB, words []string, searchByScore bool, domain []string, nodomain []string, language []string) []types.PageData {
	var args []interface{}

	wordlist := []string{"1"}
	if len(words) > 0 && words[0] != "" {
		wordlist = make([]string, 0)
		for _, word := range words {
			wordlist = append(wordlist, "word = ?")
			args = append(args, strings.ToLower(word))
		}
	}

	// the domains conditional defaults to just 'true' i.e. no domain condition
	domains := []string{"1"}
	if len(domain) > 0 && domain[0] != "" {
		domains = make([]string, 0) // we've got at least one domain! clear domains default
		for _, d := range domain {
			domains = append(domains, "domain = ?")
			args = append(args, d)
		}
	}

	nodomains := []string{"1"}
	if len(nodomain) > 0 && nodomain[0] != "" {
		nodomains = make([]string, 0)
		for _, d := range nodomain {
			nodomains = append(nodomains, "domain != ?")
			args = append(args, d)
		}
	}

	//This needs some wildcard support …
	languages := []string{"1"}
	if len(language) > 0 && language[0] != "" {
		languages = make([]string, 0)
		for _, d := range language {
			// Do a little check to avoid the database being DOSed
			if languageCodeSanityRegex.MatchString(d) {
				languages = append(languages, "lang LIKE ?")
				args = append(args, d+"%")
			}
		}
	}

	orderType := "SUM(score)"
	if !searchByScore {
		orderType = "COUNT(*)"
	}

	query := fmt.Sprintf(`
    SELECT p.url, p.about, p.title 
    FROM inv_index inv INNER JOIN pages p ON inv.url = p.url 
    WHERE (%s)
    AND (%s)
    AND (%s)
    AND (%s)
    GROUP BY inv.url 
    ORDER BY %s
    DESC
    LIMIT 15
    `, strings.Join(wordlist, " OR "), strings.Join(domains, " OR "), strings.Join(nodomains, " AND "), strings.Join(languages, " OR "), orderType)

	stmt, err := db.Prepare(query)
	util.Check(err)
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	util.Check(err)
	defer rows.Close()

	var pageData types.PageData
	var pages []types.PageData
	for rows.Next() {
		if err := rows.Scan(&pageData.URL, &pageData.About, &pageData.Title); err != nil {
			log.Fatalln(err)
		}
		pages = append(pages, pageData)
	}
	return pages
}

func InsertManyDomains(db *sql.DB, pages []types.PageData) {
	if len(pages) == 0 {
		return
	}
	values := make([]string, 0, len(pages))
	args := make([]interface{}, 0, len(pages))

	for _, b := range pages {
		values = append(values, "(?)")
		u, err := url.Parse(b.URL)
		util.Check(err)
		args = append(args, u.Hostname())
	}

	stmt := fmt.Sprintf(`INSERT OR IGNORE INTO domains(domain) VALUES %s`, strings.Join(values, ","))
	_, err := db.Exec(stmt, args...)
	util.Check(err)
}

func InsertManyPages(db *sql.DB, pages []types.PageData) {
	if len(pages) == 0 {
		return
	}
	values := make([]string, 0, len(pages))
	args := make([]interface{}, 0, len(pages))

	for _, b := range pages {
		// url, title, lang, about, domain
		values = append(values, "(?, ?, ?, ?, ?)")
		u, err := url.Parse(b.URL)
		util.Check(err)
		args = append(args, b.URL, b.Title, b.Lang, b.About, u.Hostname())
	}

	stmt := fmt.Sprintf(`INSERT OR IGNORE INTO pages(url, title, lang, about, domain) VALUES %s`, strings.Join(values, ","))
	_, err := db.Exec(stmt, args...)
	util.Check(err)
}

func InsertManyWords(db *sql.DB, batch []types.SearchFragment) {
	if len(batch) == 0 {
		return
	}

	values := make([]string, 0, len(batch))
	args := make([]interface{}, 0, len(batch))

	for _, b := range batch {
		pageurl := strings.TrimSuffix(b.URL, "/")
		values = append(values, "(?, ?, ?)")
		args = append(args, b.Word, pageurl, b.Score)
	}

	stmt := fmt.Sprintf(`INSERT OR IGNORE INTO inv_index(word, url, score) VALUES %s`, strings.Join(values, ","))
	_, err := db.Exec(stmt, args...)
	util.Check(err)
}

func InsertManyExternalLinks(db *sql.DB, externalLinks []string) {
	if len(externalLinks) == 0 {
		return
	}

	values := make([]string, 0, len(externalLinks))
	args := make([]interface{}, 0, len(externalLinks))

	for _, externalLink := range externalLinks {
		values = append(values, "(?)")
		args = append(args, externalLink)
	}

	stmt := fmt.Sprintf(`INSERT OR IGNORE INTO external_links(url) VALUES %s`, strings.Join(values, ","))
	_, err := db.Exec(stmt, args...)
	util.Check(err)
}

func InsertManyBigParagraphs(db *sql.DB, paragraphPairs []types.WholeParagraph) {
	if len(paragraphPairs) == 0 {
		return
	}

	values := make([]string, 0, len(paragraphPairs))
	args := make([]interface{}, 0, len(paragraphPairs))

	for _, paragraphPair := range paragraphPairs {
		values = append(values, "(?, ?)")
		args = append(args, paragraphPair.Text, paragraphPair.URL)
	}

	stmt := fmt.Sprintf(`INSERT OR IGNORE INTO big_search(text, url) VALUES %s`, strings.Join(values, ","))
	_, err := db.Exec(stmt, args...)
	util.Check(err)
}
