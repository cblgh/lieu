package main

import (
	"bufio"
	"fmt"
	"gomod.cblgh.org/lieu/crawler"
	"gomod.cblgh.org/lieu/database"
	"gomod.cblgh.org/lieu/ingest"
	"gomod.cblgh.org/lieu/server"
	"gomod.cblgh.org/lieu/util"
	"os"
	"strings"
)

const help = `Lieu: neighbourhood search engine

Commands
- precrawl  (scrapes config's general.url for a list of links: <li> elements containing an anchor <a> tag)
- crawl     (start crawler, crawls all urls in config's crawler.webring file. outputs to stdout)
- ingest    (ingest crawled data, generates database)
- search    (interactive cli for searching the database)
- host      (hosts search engine over http) 

Example:
    lieu precrawl > data/webring.txt 
    lieu crawl > data/source.txt
    lieu ingest
    lieu host

See the configuration file lieu.toml or 
https://github.com/cblgh/lieu for more information.
`

func main() {
	exists := util.CheckFileExists("lieu.toml")
	if !exists {
		fmt.Println("lieu: can't find config, saving an example config in the working directory")
		util.WriteMockConfig()
		fmt.Println("lieu: lieu.toml written to disk")
		util.Exit()
	}
	config := util.ReadConfig()

	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	} else {
		cmd = "help"
	}

	switch cmd {
	case "help":
		fmt.Println(help)
	case "precrawl":
		if config.General.URL == "https://example.com/" {
			fmt.Println("lieu: the url is not set (example.com)")
			util.Exit()
		}
		crawler.Precrawl(config)
	case "crawl":
		if exists := util.CheckFileExists(config.Crawler.Webring); !exists {
			fmt.Printf("lieu: webring file %q does not exist\n", config.Crawler.Webring)
			util.Exit()
		}
		sourceLen := len(util.ReadList(config.Crawler.Webring, "\n"))
		if sourceLen == 0 {
			fmt.Printf("lieu: nothing to crawl; the webring file %q is empty\n", config.Crawler.Webring)
			util.Exit()
		}
		crawler.Crawl(config)
	case "ingest":
		if exists := util.CheckFileExists(config.Data.Source); !exists {
			fmt.Printf("lieu: data source %s does not exist\n", config.Data.Source)
			fmt.Println("lieu: try running `lieu crawl`")
			util.Exit()
		}
		if sourceLen := len(util.ReadList(config.Data.Source, "\n")); sourceLen == 0 {
			fmt.Printf("lieu: nothing to ingest; data source %q is empty\n", config.Data.Source)
			fmt.Println("lieu: try running `lieu crawl`")
			util.Exit()
		}
		fmt.Println("lieu: creating a new database & initiating ingestion")
		ingest.Ingest(config)
	case "search":
		if exists := util.CheckFileExists(config.Data.Database); !exists {
			util.DatabaseDoesNotExist(config.Data.Database)
		}
		interactiveMode(config.Data.Database)
	case "random":
		if exists := util.CheckFileExists(config.Data.Database); !exists {
			util.DatabaseDoesNotExist(config.Data.Database)
		}
		db := database.InitDB(config.Data.Database)
		fmt.Println(database.GetRandomPage(db))
	case "host":
		if exists := util.CheckFileExists(config.Data.Database); !exists {
			util.DatabaseDoesNotExist(config.Data.Database)
		}
		if open := util.CheckPortOpen(config.General.Port); !open {
			fmt.Printf("lieu: port %d is not open; try another one\n", config.General.Port)
			util.Exit()
		}
		server.Serve(config)
	default:
		fmt.Println("Lieu: no such command, currently. Try `lieu help`")
	}
}

func interactiveMode(databasePath string) {
	db := database.InitDB(databasePath)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("> ")
		input, err := reader.ReadString('\n')
		util.Check(err)
		input = strings.TrimSuffix(input, "\n")
		pages := database.SearchWordsByScore(db, util.Inflect(strings.Fields(input)))
		for _, pageData := range pages {
			fmt.Println(pageData.URL)
			if len(pageData.About) > 0 {
				fmt.Println(pageData.About)
			}
		}
	}
}
