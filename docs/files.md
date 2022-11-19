# Files
_what the purposes are of all those damn files_

Lieu is based on a few files, which in turn configure various behaviours in the
**crawler** (visits urls & extracts relevant elements) and the **ingester**
(converts the crawled source data into database fields). The basic reason is to
minimize hardcoded assumptions in the source, furthering Lieu's reuse.

Below, I will refer to the files by their config defined names. Here's the
config example from the [README](../README.md), again.

```toml
[general]
name = "Merveilles Webring"
# used by the precrawl command and linked to in /about route
url = "https://webring.xxiivv.com"
port = 10001

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
```

## HTML
Before we start, a final note on some other types of files in use. The HTML
templates, used when presenting the search engine in the browser, are all
available in the [`html`](../html) folder. The includes—currently only css
& font files—are available in [`html/assets`](../html/assets).

## `[crawler]`
#### `webring`
Defines which domains will be crawled for pages. At current writing, no domains
outside of this file will be crawled.

You can populate the `webring` file manually or by precrawling an existing
webpage that contains all of the domains you want to crawl:

    lieu precrawl > data/webring.txt

#### `bannedDomains`
A list of domains that will not be crawled. This means that if they are present
in the `webring` file, they will be skipped over as candidates for crawling.

The rationale is that some of the domains of a webring may be unsuitable for ingestion
into the database. I typically find this is the case for domains that include
microblogs with 100s or 1000s of one line pages—needlessly gunking up the search
results without providing anything of interest outside the individual creating
the logs.

#### `bannedSuffixes`
Eliminates html links that end with suffixes present in this file. Typically I want
to avoid crawling links to media formats such as `.mp4`, and other types of
non-html documents, really.

It's fine to leave this file intact with its defaults.

#### `boringWords`
This file is a bit more specific. It contains words which, if present in a link,
will prevent the link from being logged. The reason is cause it suggests the
link target is boring—irrelevant for this application of the search engine.

This can be `javascript:` script links, or other types of content that is less
relevant to the focus of the search engine & webring.

Link data of this type is as yet unused in Lieu's ingestion.

#### `boringDomains`
Like `boringWords` except it contains a list of domains which are banned from
having their links be logged, typically because they are deemed less relevant
for the focus of the search engine.

Link data of this type is as yet unused in Lieu's ingestion.

## `[data]`
#### `source`
Contains the linewise data that was produced by the crawler. The first word
identifies the type of data and the last word identifies the page the data
originated from.

Example:
```
h2 Prelude https://cblgh.org/articles/four-nights-in-tornio.html
```

* An `<h2>` tag was scraped, 
* its contents were `Prelude`, and 
* the originating article was https://cblgh.org/articles/four-nights-in-tornio.html

#### `database`
The location the sqlite3 database will be created & read from.

#### `heuristics`
Heuristics contains a list of words or phrases which disqualify scraped
paragraphs from being used as descriptive text Lieu's search results. Typically
excluded are e.g. paragraphs which contain copyright symbols—as that indicates we
have scraped the bottom-most paragraph, i.e. the page was likely a short stub,
with a better content description elsewhere.

#### `wordlist`
Also known as [stopwords](https://en.wikipedia.org/wiki/Stop_word)—words which
are stopped from entering the search index. The default wordlist consists of the
1000 or so most common English words, albeit curated slightly to still allow for
interesting concepts and verbs—such as `reading` and `books`, for example.

#### `previewQueryList`
A list of css selectors (one per line) to fetch preview paragraphs,
the first paragraph found that passes a check against the `heuristics` file makes
it into the search index. For each selector lieu tries the first four paragraphs
found with each selector before skipping to the next one.

To get good results one usually wants to tune this to getting the first "real" paragraph
after the header, or a summary paragraph if provided. It is also worth trying to avoind getting
irelevant paragraphs as they clutter up your index and results, lieu will fall back to other
preview sources.

The default has been (at the time of writing) tuned for use with the Fediring.

Depending on how well the websites you are indexing are with semantic HTML this will
get you the 70 to 90% solution. For the rest use heuristics and contact the creators of the
websites you are tring to index, they (usually) appreciate the feedback.

#### OpenSearch metadata
If you are running your own instance of Lieu, you might want to look into changing the URL
defined in the file `opensearch.xml`, which specifies [OpenSearch
metadata](https://en.wikipedia.org/wiki/OpenSearch). This file allows a Lieu instance to be
added to any browser supporting OpenSearch as one of the search engines that can be used for
browser searches.

See [html/assets/opensearch.xml](../html/assets/opensearch.xml).
