package search

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html/charset"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/PuerkitoBio/goquery"

	logger "github.com/sirupsen/logrus"
)

type Result struct {
	Title    string
	Magnet   string
	Size     int64
	Seeders  int64
	Leechers int64
	Created  time.Time
}

var (
	// Rutracker login data
	RutrackerUser string
	RutrackerPass string
)

func init() {
	logger.SetLevel(logger.DebugLevel)
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func (r *Result) FormattedCreatedDate() string {
	return r.Created.Format("02.01.2006")
}

func SearchPirateBay(query string, page int) ([]Result, []int, error) {
	rawurl := "https://thepiratebay.org/search/" + url.QueryEscape(query) + "/" + strconv.Itoa(page-1) + "/99/200"
	logger.Debugf("piratebay: search query %s", rawurl)
	res, err := GET(rawurl, nil)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var results []Result
	pages := makeRange(1, 33) // static number
	doc.Find("#searchResult").Find("tbody").Find("tr").Each(func(i int, s *goquery.Selection) {
		td1 := s.Find("td").Eq(1)
		td2 := s.Find("td").Eq(2)
		td3 := s.Find("td").Eq(3)

		// title
		var title string
		if link := td1.Find("a.detLink"); link != nil {
			title = link.AttrOr("title", "")
			title = strings.TrimSpace(title)
			title = strings.TrimPrefix(title, "Details for ")
		}
		if title == "" {
			logger.Debugf("piratebay: no title found")
			return
		}

		// magnet
		magnet := td1.ChildrenFiltered("a").Eq(0).AttrOr("href", "")
		if magnet == "" {
			logger.Debugf("piratebay: no magnet found")
			return
		}

		// size
		var size int64
		if desc := td1.Find("font.detDesc"); desc != nil {
			if parts := strings.Split(desc.Text(), ", "); len(parts) == 3 {
				if fields := strings.Fields(parts[1]); len(fields) == 3 {
					n, err := humanize.ParseBytes(fields[1] + " " + fields[2])
					if err == nil {
						size = int64(n)
					}
				}
			}
		}
		if size == 0 {
			logger.Debugf("piratebay: no size found")
			return
		}

		// seeders
		var seeders int64
		seeders, _ = strconv.ParseInt(strings.TrimSpace(td2.Text()), 10, 64)
		if seeders == 0 {
			logger.Debugf("piratebay: no seeders found")
			return
		}

		// leechers
		var leechers int64
		leechers, _ = strconv.ParseInt(strings.TrimSpace(td3.Text()), 10, 64)

		// created
		var created time.Time
		if desc := td1.Find("font.detDesc"); desc != nil {
			if parts := strings.Split(desc.Text(), ", "); len(parts) == 3 {
				if fields := strings.Fields(parts[0]); len(fields) == 3 {
					mdy := fields[1] + " " + fields[2]
					created, err = time.Parse(`01-02 2006`, mdy)
					if err != nil {
						tmpstr := mdy + " " + strconv.Itoa(time.Now().Year())
						created, err = time.Parse(`01-02 15:04 2006`, tmpstr)
						if err != nil {
							logger.Debugf("piratebay: parsing %q failed: %s", tmpstr, err)
						}
					}
				}
			}
		}
		if created.IsZero() {
			// return
		}

		results = append(results, Result{
			Title:    title,
			Magnet:   magnet,
			Size:     size,
			Seeders:  seeders,
			Leechers: leechers,
			Created:  created,
		})
	})

	return results, pages, nil
}

func SearchRutracker(query string, page int) ([]Result, []int, error) {
	rawurl := "https://rutracker.org/forum/tracker.php?f=1105,1389,599&start=" + strconv.Itoa((page-1)*50) + "&nm=" + url.QueryEscape(query)
	logger.Debugf("rutracker: search query: %q", rawurl)

	loginUrl := "https://rutracker.org/forum/login.php"
	userForm := url.Values{
		"login_username": {RutrackerUser},
		"login_password": {RutrackerPass},
		"login":          {"%C2%F5%EE%E"},
	}
	postres, jar, err := POST(loginUrl, userForm)
	if err != nil {
		return nil, nil, err
	}
	defer postres.Body.Close()

	res, err := GET(rawurl, jar)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	contentType := res.Header.Get("Content-Type")
	utf8reader, err := charset.NewReader(res.Body, contentType)
	if err != nil {
		logger.Errorf("rutracker: cant change charset")
		return nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(utf8reader)
	if err != nil {
		return nil, nil, err
	}
	var results []Result

	// parse max pages
	maxpage := doc.Find(".bottom_info p b").Eq(1).Text()
	var numres int64
	numres = 1
	if maxpage != "" {
		numres, _ = strconv.ParseInt(strings.TrimSpace(maxpage), 10, 64)
	}
	pages := makeRange(1, int(numres))

	doc.Find("#search-results").Find("tbody").Find("tr").Each(func(i int, s *goquery.Selection) {
		td1 := s.Find("td").Eq(3) // title
		td2 := s.Find("td").Eq(5) // size
		td3 := s.Find("td").Eq(6) // S
		td4 := s.Find("td").Eq(7) // L
		td5 := s.Find("td").Eq(9) // created

		// title and id for api
		var title string
		var id string
		if link := td1.Find("a"); link != nil {
			id = link.AttrOr("data-topic_id", "")
			title = link.Text()
		}
		if len(title) == 0 {
			logger.Debugf("rutracker: no title found")
			return
		}
		if id == "" {
			logger.Debugf("rutracker: no id found")
			return
		}

		// magnet
		apiurl := "http://api.rutracker.org/v1/get_tor_hash?by=topic_id&val=" + id
		apires, err := GET(apiurl, nil)
		if err != nil {
			logger.Debugf("rutracker api: failed to connect")
			logger.Debugf("rutracker api: %q", err)
		}
		defer apires.Body.Close()

		var r map[string]map[string]interface{}
		var magnet string
		body, err := ioutil.ReadAll(apires.Body)
		if err := json.Unmarshal(body, &r); err != nil {
			logger.Debugf("rutracker api: cant parse response: %s : %s", body, err)
		}
		magnet = "magnet:?xt=urn:btih:" + r["result"][id].(string) + "&tr=http%3A%2F%2Fbt.t-ru.org%2Fann%3Fmagnet"

		if magnet == "" {
			logger.Debugf("rutracker api: no magnet found")
			return
		}

		// size
		var size int64
		if link := td2.Find("a"); link != nil {
			if fields := strings.Fields(link.Text()); len(fields) == 3 {
				n, err := humanize.ParseBytes(fields[0] + " " + fields[1])
				if err == nil {
					size = int64(n)
				}
			}
		}
		if size == 0 {
			logger.Debugf("rutracker: no size found")
			return
		}

		// seeders
		var seeders int64
		if link := td3.Find("b"); link != nil {
			seeders, _ = strconv.ParseInt(strings.TrimSpace(link.Text()), 10, 64)
		}
		if seeders == 0 {
			logger.Debugf("rutracker: no seeders found")
			return
		}

		// leechers
		var leechers int64
		leechers, _ = strconv.ParseInt(strings.TrimSpace(td4.Text()), 10, 64)

		// created
		var created time.Time
		ts, err := strconv.ParseInt(td5.AttrOr("data-ts_text", ""), 10, 64)
		if err != nil {
			logger.Debugf("rutracker: failed to parse timestamp")
		}
		created = time.Unix(ts, 0)

		if created.IsZero() {
			// return
		}

		results = append(results, Result{
			Title:    title,
			Magnet:   magnet,
			Size:     size,
			Seeders:  seeders,
			Leechers: leechers,
			Created:  created,
		})
	})

	return results, pages, nil
}

func SearchTokioTosho(query string, page int) ([]Result, []int, error) {
	rawurl := "https://www.tokyotosho.info/search.php?&page=" + strconv.Itoa(page) + "&terms=" + url.QueryEscape(query)
	logger.Debugf("tokiotosho: search query: %q", rawurl)
	res, err := GET(rawurl, nil)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	var results []Result

	var numres int64
	if fields := strings.Split(doc.Find(".listing").Next().Text(), " "); len(fields) >= 6 {
		numres, _ = strconv.ParseInt(strings.TrimSuffix(fields[len(fields)-1], "."), 10, 64)
		numres /= 50
	}
	var title string
	var magnet string
	logger.Debugf("tokiotosho: results: %q", results)
	pages := makeRange(1, int(numres)+1)
	doc.Find(".listing").Find("tbody").Find("tr.category_0").Each(func(i int, s *goquery.Selection) {
		var size int64
		var created time.Time
		var seeders int64
		var leechers int64
		var skipnext bool

		if i%2 == 0 {
			td1 := s.Find("td").Eq(1) // title and magnet
			if link := td1.ChildrenFiltered("a").Eq(1); link != nil {
				title = strings.TrimSpace(link.Text())
			}
			if title == "" {
				logger.Debugf("tokiotosho: no title found")
				skipnext = true
				return
			}

			// magnet
			magnet = td1.ChildrenFiltered("a").Eq(0).AttrOr("href", "")
			if magnet == "" {
				skipnext = true
				logger.Debugf("tokiotosho: no magnet found")
				return
			}
		} else {
			if skipnext {
				skipnext = false
				return
			}
			td2 := s.Find("td").Eq(0) // size and date
			td3 := s.Find("td").Eq(1) // S and L

			if parts := strings.Split(td2.Text(), " | "); len(parts) >= 3 {
				// size
				if fields := strings.Fields(parts[1]); len(fields) >= 2 {
					n, err := humanize.ParseBytes(fields[1])
					if err == nil {
						size = int64(n)
					}
				}
				// created
				if fields := strings.Fields(parts[2]); len(fields) >= 4 {
					t := strings.Join(fields[1:], " ")
					created, err = time.Parse(`2006-01-02 15:04 MST`, t)

					if err != nil {
						logger.Debugf("tokiotosho: parsing %q failed: %s", t, err)
					}
				}

			}

			if created.IsZero() {
				logger.Debugf("tokiotosho: no created time found")
				// return
			}
			if size == 0 {
				logger.Debugf("tokiotosho: no size found")
				return
			}

			// seeders and leechers

			if parts := strings.Split(td3.Text(), " "); len(parts) >= 4 {
				seeders, _ = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				leechers, _ = strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
			}
			if seeders == 0 {
				logger.Debugf("tokiotosho: no seeders found")
				return
			}

			results = append(results, Result{
				Title:    title,
				Magnet:   magnet,
				Size:     size,
				Seeders:  seeders,
				Leechers: leechers,
				Created:  created,
			})
		}
	})

	return results, pages, nil
}

/*func rutrackerCookies() (*cookiejar.Jar, error) {
	loginUrl := "https://rutracker.org/forum/login.php"
	userForm := url.Values{
		"login_username": {RutrackerUser},
		"login_password": {RutrackerPass},
		"login":          {"%C2%F5%EE%E"},
	}
	res, err := POST(loginUrl, userForm)
	if err != nil {
		return nil, err
	}
	logger.Debugf("rutracker api: login credits: %s", userForm)
	logger.Debugf("rutracker api: cookies: %s", res.Cookies())
	logger.Debugf("rutracker api: login info: %q", res)
	jar, _ := cookiejar.New(nil)
	var cookies []*http.Cookie

	for _, cookie := range res.Cookies() {
		switch cookie.Name {
		case "bb_session", "bb_guid", "bb_t":
			cookies = append(cookies, cookie)
			logger.Debugf("rutracker api cookies: %s : %s", cookie.Name, cookie.Value)
		}
	}
	parsedUrl, _ := url.Parse(loginUrl)
	jar.SetCookies(parsedUrl, cookies)
	return jar, nil
}*/

func GET(rawurl string, jar *Jar) (*http.Response, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	if jar != nil {
		httpClient.Jar = jar
	}
	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: %s", res.Status)
	}

	return res, nil
}

func POST(rawurl string, form url.Values) (*http.Response, *Jar, error) {
	jar := NewJar()
	client := &http.Client{
		Jar: jar,
	}
	res, err := client.PostForm(
		rawurl,
		form,
	)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("request failed: %s", res.Status)
	}
	return res, jar, nil
}

type Jar struct {
	sync.Mutex
	cookies map[string][]*http.Cookie
}

func NewJar() *Jar {
	jar := new(Jar)
	jar.cookies = make(map[string][]*http.Cookie)
	return jar
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.Lock()
	if _, ok := jar.cookies[u.Host]; ok {
		for _, c := range cookies {
			jar.cookies[u.Host] = append(jar.cookies[u.Host], c)
		}
	} else {
		jar.cookies[u.Host] = cookies
	}
	jar.Unlock()
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies[u.Host]
}
