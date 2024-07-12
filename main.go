package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	urllib "net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Page struct {
	url   string
	path  string
	depth int
}

type Crawler struct {
	root    string
	workers int
	depth   int
	client  http.Client
	seen    SafeMap
	queue   chan Page
	sem     chan bool
	wg      sync.WaitGroup

	countMut sync.Mutex
	count    int
}

func newCrawler(r string, w int, d int, t int) *Crawler {
	return &Crawler{
		root:    r,
		workers: w,
		depth:   d,
		client: http.Client{
			Timeout: time.Duration(t) * time.Second,
		},
		seen:  *NewSafeMap(),
		queue: make(chan Page),
		sem:   make(chan bool, w),
	}
}

func (c *Crawler) worker() {
	for page := range c.queue {
		func() {
			defer c.wg.Done()

			if page.depth > c.depth {
				return
			}

			log.Printf("visiting depth %d page %s\n", page.depth, page.url)

			req, err := http.NewRequest("GET", page.url, nil)
			if err != nil {
				log.Printf("failed building get request %s -> %s\n", page.url, err)
				return
			}

			etag, err := CheckEtag(page.path)
			if err != nil {
				log.Printf("failed reading etag %s -> %s\n", page.url, err)
				return
			}

			req.Header.Add("If-None-Match", etag)
			r, modified, err := c.fetchContent(req)
			if err != nil {
				log.Printf("failed fetching url %s -> %s\n", page.url, err)
				return
			}

			if !modified {
				log.Printf("unmodified %s\n", page.url)
				return
			}

			defer r.Body.Close()

			mediatype, err := getMediaType(r)
			if err != nil {
				log.Printf("failed parsing mediatype for %s -> %s\n", page.url, err)
				return
			}

			if mediatype != "text/html" {
				if err = SaveEtag(page.path, r.Header.Get("Etag")); err != nil {
					log.Printf("failed saving etag %s -> %s\n", page.url, err)
				}

				if err = saveContent(page, r); err != nil {
					log.Printf("error saving contents of %s depth %d -> %v\n", page.url, page.depth, err)
				} else {
					c.countMut.Lock()
					c.count++
					c.countMut.Unlock()
				}

				return
			}

			err = os.Mkdir(page.path, os.FileMode(0755))
			if err != nil {
				if !errors.Is(err, os.ErrExist) && page.path != "" {
					log.Printf("error creating folder %s -> %v\n", page.path, err)
					return
				}
			}

			hrefs, err := extractHrefs(r)
			if err != nil {
				log.Printf("failed extracting hrefs from %s -> %s\n", page.url, err)
				return
			}

			for _, href := range hrefs {
				newUrl := page.url + href

				// if already seen url
				if !c.seen.ConditionalInsert(newUrl) {
					continue
				}

				newPath, err := urllib.QueryUnescape(page.path + href)
				if err != nil {
					return
				}

				c.sem <- true
				c.wg.Add(1)
				go func() {
					c.queue <- Page{
						url:   newUrl,
						path:  newPath,
						depth: page.depth + 1,
					}
				}()
				<-c.sem
			}
		}()
	}
}

func (c *Crawler) fetchContent(req *http.Request) (*http.Response, bool, error) {
	r, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}

	if r.StatusCode == http.StatusNotModified {
		r.Body.Close()
		return nil, false, nil
	}

	if r.StatusCode != http.StatusOK {
		r.Body.Close()
		return nil, false, fmt.Errorf("non 200 status code, %d %s", r.StatusCode, r.Status)
	}

	return r, true, nil
}

func getMediaType(r *http.Response) (string, error) {
	content_type := r.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(content_type)
	if err != nil {
		return "", fmt.Errorf("invalid content type header -> %s", err)
	}

	return mediatype, nil
}

func saveContent(p Page, r *http.Response) error {
	file, err := os.OpenFile(p.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error opening file %s -> %v", p.path, err)
	}

	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		return fmt.Errorf("error writing to file %s -> %v", p.path, err)
	}

	return nil
}

func extractHrefs(r *http.Response) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed parsing page content -> %s", err)
	}

	hrefs := []string{}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}

		if saneHref, valid := SanitizeRelativeHref(href); valid {
			hrefs = append(hrefs, saneHref)
		}
	})

	return hrefs, nil
}

var prefixFilters = []string{
	"mailto:", "tel:", "javascript:",
}

func SanitizeRelativeHref(href string) (string, bool) {
	if len(href) == 0 {
		return "", false
	}

	if href[0] == '#' || href[0] == '?' {
		return "", false
	}

	// discard current paths
	if href == "/" || href == "." || href == "./" {
		return "", false
	}

	// trim current directory from path if necessary
	href = strings.TrimPrefix(href, "./")

	// only accept forwarding href
	if strings.HasPrefix(href, "..") {
		return "", false
	}

	parsed, err := urllib.Parse(href)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return "", false
	}

	for _, prefix := range prefixFilters {
		if strings.HasPrefix(href, prefix) {
			return "", false
		}
	}

	if href[0] == '/' {
		href = href[1:]
	}

	return href, true
}

func main() {
	var (
		root    = flag.String("r", "http://localhost:8000/", "file server root url")
		workers = flag.Int("w", 10, "number of workers (maximum concurrent HTTP requests)")
		depth   = flag.Int("d", 20, "maximum crawl depth")
		timeout = flag.Int("t", 20, "HTTP requests timeout in seconds")
	)
	flag.Parse()

	MustInitDB(*root)

	c := newCrawler(*root, *workers, *depth, *timeout)
	parsed, err := urllib.Parse(c.root)
	if err != nil {
		log.Fatalf("error parsing root url %s -> %s\n", c.root, err)
	}
	err = os.Mkdir(parsed.Host, os.FileMode(0755))
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Fatalf("failed creating root directory %s -> %s\n", parsed.Host, err)
		}
	}
	err = os.Chdir(parsed.Host)
	if err != nil {
		log.Fatalf("error changing working directory into %s -> %s\n", parsed.Host, err)
	}

	for range c.workers {
		go c.worker()
	}

	c.wg.Add(1)
	c.seen.Insert(c.root)
	c.queue <- Page{
		url:   c.root,
		path:  "",
		depth: 1,
	}

	start := time.Now()
	c.wg.Wait()
	close(c.queue)
	took := time.Since(start)
	fmt.Printf("completed, fetched %d files in %02d:%02d:%02d\n", c.count, int(took.Hours()), int(took.Minutes())%60, int(took.Seconds())%60)
}
