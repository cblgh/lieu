package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
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

type TemplateView struct {
	SiteName string
	Data     interface{}
}

type SearchData struct {
	Query      string
	Title      string
	Site       string
	Pages      []types.PageData
	IsInternal bool
}

type IndexData struct {
	Tagline     string
	Placeholder string
}

type ListData struct {
	Title string
	URLs  []types.PageData
}

type AboutData struct {
	DomainCount  int
	WebringName  string
	PageCount    string
	TermCount    string
	FilteredLink string
	RingLink     string
}

var templates = template.Must(template.ParseFiles(
	"html/head.html", "html/nav.html", "html/footer.html",
	"html/about.html", "html/index.html", "html/list.html", "html/search.html", "html/webring.html"))

const useURLTitles = true

func (h RequestHandler) searchRoute(res http.ResponseWriter, req *http.Request) {
	var query string
	view := &TemplateView{}

	var domain string
	if req.Method == http.MethodGet {
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
			query = words[0]
		}

		// how to use: https://gist.github.com/cblgh/29991ba0a9e65cccbe14f4afd7c975f1
		if parts, exists := params["site"]; exists && parts[0] != "" {
			// make sure we only have the domain, and no protocol prefix
			domain = strings.TrimPrefix(parts[0], "https://")
			domain = strings.TrimPrefix(domain, "http://")
			domain = strings.TrimSuffix(domain, "/")
		}
	}

	if len(query) == 0 {
		view.Data = IndexData{Tagline: h.config.General.Tagline, Placeholder: h.config.General.Placeholder}
		h.renderView(res, "index", view)
		return
	}

	var pages []types.PageData
	if domain != "" {
		pages = database.SearchWordsBySite(h.db, util.Inflect(strings.Fields(query)), domain)
	} else {
		pages = database.SearchWordsByScore(h.db, util.Inflect(strings.Fields(query)))
	}

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Title:      "Results",
		Query:      query,
		Site:       domain,
		Pages:      pages,
		IsInternal: true,
	}
	h.renderView(res, "search", view)
}

func (h RequestHandler) externalSearchRoute(res http.ResponseWriter, req *http.Request) {
	var query string
	view := &TemplateView{}

	if req.Method == http.MethodGet {
		params := req.URL.Query()
		if words, exists := params["q"]; exists && words[0] != "" {
			query = words[0]
		}
	}

	pages := database.FulltextSearchWords(h.db, query)

	if useURLTitles {
		for i, pageData := range pages {
			prettyURL, err := url.QueryUnescape(strings.TrimPrefix(strings.TrimPrefix(pageData.URL, "http://"), "https://"))
			util.Check(err)
			pageData.Title = prettyURL
			pages[i] = pageData
		}
	}

	view.Data = SearchData{
		Title:      "External Results",
		Query:      query,
		Pages:      pages,
		IsInternal: false,
	}
	h.renderView(res, "search", view)
}

func (h RequestHandler) aboutRoute(res http.ResponseWriter, req *http.Request) {
	view := &TemplateView{}

	pageCount := util.Humanize(database.GetPageCount(h.db))
	wordCount := util.Humanize(database.GetWordCount(h.db))
	domainCount := database.GetDomainCount(h.db)

	view.Data = AboutData{
		WebringName:  h.config.General.Name,
		DomainCount:  domainCount,
		PageCount:    pageCount,
		TermCount:    wordCount,
		FilteredLink: "/filtered",
		RingLink:     h.config.General.URL,
	}
	h.renderView(res, "about", view)
}

func (h RequestHandler) filteredRoute(res http.ResponseWriter, req *http.Request) {
	view := &TemplateView{}

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

	view.Data = ListData{
		Title: "Filtered Domains",
		URLs:  URLs,
	}
	h.renderView(res, "list", view)
}

func (h RequestHandler) randomRoute(res http.ResponseWriter, req *http.Request) {
	link := database.GetRandomPage(h.db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func (h RequestHandler) randomExternalRoute(res http.ResponseWriter, req *http.Request) {
	link := database.GetRandomExternalLink(h.db)
	http.Redirect(res, req, link, http.StatusSeeOther)
}

func (h RequestHandler) webringRoute(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, h.config.General.URL, http.StatusSeeOther)
}

func (h RequestHandler) renderView(res http.ResponseWriter, tmpl string, view *TemplateView) {
	view.SiteName = h.config.General.Name
	var errTemp error
	if _, exists := os.LookupEnv("LIEU_DEV"); exists {
		var templates = template.Must(template.ParseFiles(
			"html/head.html", "html/nav.html", "html/footer.html",
			"html/about.html", "html/index.html", "html/list.html", "html/search.html", "html/webring.html"))
		errTemp = templates.ExecuteTemplate(res, tmpl+".html", view)
	} else {
		errTemp = templates.ExecuteTemplate(res, tmpl+".html", view)
	}
	util.Check(errTemp)
}

func Serve(config types.Config) {
	db := database.InitDB(config.Data.Database)
	handler := RequestHandler{config: config, db: db}

	http.HandleFunc("/about", handler.aboutRoute)
	http.HandleFunc("/", handler.searchRoute)
	http.HandleFunc("/outgoing", handler.externalSearchRoute)
	http.HandleFunc("/random/outgoing", handler.randomExternalRoute)
	http.HandleFunc("/random", handler.randomRoute)
	http.HandleFunc("/webring", handler.webringRoute)
	http.HandleFunc("/filtered", handler.filteredRoute)

	fileserver := http.FileServer(http.Dir("html/assets/"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fileserver))

	portstr := fmt.Sprintf(":%d", config.General.Port)
	fmt.Println("Listening on port: ", portstr)

	http.ListenAndServe(portstr, nil)
}
