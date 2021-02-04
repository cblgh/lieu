package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
    "database/sql"

	"lieu/database"
	"lieu/types"
	"lieu/util"
	"html/template"

    // "github.com/shurcooL/vfsgen"
)

type SearchData struct {
	Query string
	Pages []types.PageData
}

type AboutData struct {
	DomainCount  int
	InstanceName string
	PageCount    string
	TermCount    string
	FilteredLink string
	RingLink     string
}

const useURLTitles = true

func searchRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	var query string

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		words, exists := params["q"]
		if !exists || words[0] == "" {
			view := template.Must(template.ParseFiles("html/index-template.html"))
			var empty interface{}
            err := view.Execute(res, empty)
            util.Check(err)
			return
		}
		query = words[0]
	} else {
		view := template.Must(template.ParseFiles("html/index-template.html"))
		var empty interface{}
		err := view.Execute(res, empty)
        util.Check(err)
		return
	}

	pages := database.SearchWordsByScore(db, util.Inflect(strings.Fields(query)))

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view := template.Must(template.ParseFiles("html/search-template.html"))
	data := SearchData{
		Query: query,
		Pages: pages,
	}
	err := view.Execute(res, data)
    util.Check(err)
}

func aboutRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	pageCount := util.Humanize(database.GetPageCount(db))
	wordCount := util.Humanize(database.GetWordCount(db))
	domainCount := database.GetDomainCount(db)

	view := template.Must(template.ParseFiles("html/about-template.html"))
	data := AboutData{
		InstanceName: config.General.Name,
		DomainCount:  domainCount,
		PageCount:    pageCount,
		TermCount:    wordCount,
		FilteredLink: "/filtered",
		RingLink:     config.General.URL,
	}
	err := view.Execute(res, data)
    util.Check(err)
}

type ListData struct {
	Title string
	URLs  []types.PageData
}

func filteredRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
	view := template.Must(template.ParseFiles("html/list-template.html"))
	var URLs []types.PageData
	for _, domain := range util.ReadList(config.Crawler.BannedDomains, "\n") {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		u.Scheme = "https"
		p := types.PageData{Title: domain, URL: u.String()}
		URLs = append(URLs, p)
	}
	data := ListData{
		Title: "Filtered Domains",
		URLs:  URLs,
	}
	err := view.Execute(res, data)
    util.Check(err)
}

func randomRoute(res http.ResponseWriter, req *http.Request, config types.Config, db *sql.DB) {
    link := database.GetRandomPage(db)
    http.Redirect(res, req, link, http.StatusSeeOther)
}

func Serve(config types.Config) {
	db := database.InitDB(config.Data.Database)

	http.HandleFunc("/about", func(res http.ResponseWriter, req *http.Request) {
		aboutRoute(res, req, config, db)
	})
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		searchRoute(res, req, config, db)
	})

    http.HandleFunc("/random", func(res http.ResponseWriter, req *http.Request) {
        randomRoute(res, req, config, db)
    })

	http.HandleFunc("/filtered", func(res http.ResponseWriter, req *http.Request) {
		filteredRoute(res, req, config, db)
	})

	fileserver := http.FileServer(http.Dir("html/assets/"))
	http.Handle("/links/", http.StripPrefix("/links/", fileserver))

	portstr := fmt.Sprintf(":%d", config.General.Port)
	fmt.Println("listening on", portstr)

	http.ListenAndServe(portstr, nil)
}
