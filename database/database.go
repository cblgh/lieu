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
	"log"
	"net/url"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

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
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Fatalln(err)
		}
	}
}

/* TODO: filters
lang:en|fr|en|<..>
site:wiki.xxiivv.com, site:cblgh.org
nosite:excluded-domain.com

"word1 word2 word3" strict query

query params:
&order=score, &order=count
&outgoing=true
*/

func SearchWordsByScore(db *sql.DB, words []string) []types.PageData {
	return searchWords(db, words, true)
}

func SearchWordsByCount(db *sql.DB, words []string) []types.PageData {
	return searchWords(db, words, false)
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

func GetRandomPage(db *sql.DB) string {
    rows, err := db.Query("SELECT url FROM pages ORDER BY RANDOM() LIMIT 1;")
    util.Check(err)

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
	var count int
	for rows.Next() {
		err = rows.Scan(&count)
		util.Check(err)
	}
	return count
}

func searchWords(db *sql.DB, words []string, searchByScore bool) []types.PageData {
	var wordlist []string
	var args []interface{}
	for _, word := range words {
		wordlist = append(wordlist, "word = ?")
		args = append(args, strings.ToLower(word))
	}

	orderType := "SUM(score)"
	if !searchByScore {
		orderType = "COUNT(*)"
	}

	query := fmt.Sprintf(`
    SELECT p.url, p.about, p.title 
    FROM inv_index inv INNER JOIN pages p ON inv.url = p.url 
    WHERE %s
    GROUP BY inv.url 
    ORDER BY %s
    DESC
    LIMIT 15
    `, strings.Join(wordlist, " OR "), orderType)

	stmt, err := db.Prepare(query)
	util.Check(err)
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	util.Check(err)
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
