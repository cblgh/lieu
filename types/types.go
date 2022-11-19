package types

type SearchFragment struct {
	Word  string
	URL   string
	Score int
}

type PageData struct {
	URL   string
	Title string
	About string
	Lang  string
	AboutSource string
}

type Config struct {
	General struct {
		Name            string `json:name`
		Tagline         string `json:tagline`
		Placeholder     string `json:placeholder`
		URL             string `json:url`
		WebringSelector string `json:"webringSelector"`
		Port            int    `json:port`
		Proxy           string `json:proxy`
	} `json:general`
	Theme struct {
		Foreground string `json:"foreground"`
		Background string `json:"background"`
		Links      string `json:"links"`
	} `json:"theme"`
	Data struct {
		Source         string `json:source`
		Database       string `json:database`
		Heuristics     string `json:heuristics`
		Wordlist       string `json:wordlist`
	} `json:data`
	Crawler struct {
		Webring        string `json:webring`
		BannedDomains  string `json:bannedDomains`
		BannedSuffixes string `json:bannedSuffixes`
		BoringWords    string `json:boringWords`
		BoringDomains  string `json:boringDomains`
		PreviewQueries string `json:"previewQueryList"`
	} `json:crawler`
}
