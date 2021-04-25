package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"html/template"
	"lieu/database"
	"lieu/types"
	"lieu/util"
)

type RequestHandler struct {
	config types.Config
	db     *sql.DB
}

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

type ListData struct {
	Title string
	URLs  []types.PageData
}

const useURLTitles = true

var indexView = template.Must(template.ParseFiles("html/index-template.html"))
var aboutView = template.Must(template.ParseFiles("html/about-template.html"))
var listView = template.Must(template.ParseFiles("html/list-template.html"))
var searchResultsView = template.Must(template.ParseFiles("html/search-template.html"))

func (h RequestHandler) searchRoute(res http.ResponseWriter, req *http.Request) {
	var query string

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
            query = words[0]
        }
	}

    if len(query) == 0 {
		var empty interface{}
		err := indexView.Execute(res, empty)
		util.Check(err)
		return
	}

	pages := database.SearchWordsByScore(h.db, util.Inflect(strings.Fields(query)))

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	data := SearchData{
		Query: query,
		Pages: pages,
	}
	err := searchResultsView.Execute(res, data)
	util.Check(err)
}

func (h RequestHandler) aboutRoute(res http.ResponseWriter, req *http.Request) {
	pageCount := util.Humanize(database.GetPageCount(h.db))
	wordCount := util.Humanize(database.GetWordCount(h.db))
	domainCount := database.GetDomainCount(h.db)

	data := AboutData{
		InstanceName: h.config.General.Name,
		DomainCount:  domainCount,
		PageCount:    pageCount,
		TermCount:    wordCount,
		FilteredLink: "/filtered",
		RingLink:     h.config.General.URL,
	}
	err := aboutView.Execute(res, data)
	util.Check(err)
}

func (h RequestHandler) filteredRoute(res http.ResponseWriter, req *http.Request) {
	var URLs []types.PageData
	for _, domain := range util.ReadList(h.config.Crawler.BannedDomains, "\n") {
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
	err := listView.Execute(res, data)
	util.Check(err)
}

func (h RequestHandler) randomRoute(res http.ResponseWriter, req *http.Request) {
	link := database.GetRandomPage(h.db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func Serve(config types.Config) {
	db := database.InitDB(config.Data.Database)
	handler := RequestHandler{config: config, db: db}

	http.HandleFunc("/about", handler.aboutRoute)
	http.HandleFunc("/", handler.searchRoute)
	http.HandleFunc("/random", handler.randomRoute)
	http.HandleFunc("/filtered", handler.filteredRoute)

	fileserver := http.FileServer(http.Dir("html/assets/"))
	http.Handle("/links/", http.StripPrefix("/links/", fileserver))

	portstr := fmt.Sprintf(":%d", config.General.Port)
	fmt.Println("listening on", portstr)

	http.ListenAndServe(portstr, nil)
}
