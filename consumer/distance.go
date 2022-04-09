package main

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
)

var mu sync.Mutex

func findDistance(abs_source, abs_target string) ([]string, error) {
	_, source, _ := strings.Cut(abs_source, "wikipedia.org")
	_, target, _ := strings.Cut(abs_target, "wikipedia.org")

	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(20),
		// colly.Debugger(&debug.LogDebugger{}),
		colly.AllowURLRevisit(),
	)
	c.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})
	c.Limit(&colly.LimitRule{
		DomainRegexp: "https://[a-z]+\\.wikipedia\\.org/*.",
		Parallelism:  2,
		Delay:        3 * time.Second,
		RandomDelay:  15 * time.Second,
	})
	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	// // errCount := make(map[colly.Request]uint)
	// c.OnError(func(r *colly.Response, err error) {
	// 	log.Println("Request URL:", r.Request.URL, "failed with response:", *r, "\nError:", err)
	// 	// if c, ok := errCount[*r.Request]; r.StatusCode != 200 && ok && c < 3 {
	// 	// 	if !ok {
	// 	// 		c = 0
	// 	// 	}
	// 	// 	errCount[*r.Request] = c + 1
	// 	// 	time.Sleep(1 * time.Second)
	// 	// 	r.Request.Retry()
	// 	// }
	// })

	depths := make(map[string]int)
	depths[source] = 0
	depths[target] = 100000
	froms := make(map[string]string)

	re := regexp.MustCompile(`/wiki/[^:]+$`)

	c.OnHTML(".mw-parser-output", func(e *colly.HTMLElement) {
		depth := e.Request.Depth
		from := e.Request.URL.Path
		links := e.ChildAttrs("a", "href")

		found := false
		for _, link := range links {
			found = found || (link == target)
		}

		mu.Lock()
		depth_target := depths[target]
		mu.Unlock()

		if found && depth < depth_target {
			mu.Lock()
			depths[target] = depth
			froms[target] = from
			mu.Unlock()

			return
		}

		idxs := []int{}
		for i, link := range links {
			matched := re.MatchString(link)
			if !matched {
				continue
			}

			mu.Lock()
			if _, ok := depths[link]; ok {
				if depth < depths[link] {
					depths[link] = depth
					froms[link] = from
					mu.Unlock()
				} else {
					mu.Unlock()
					continue
				}
			} else {
				depths[link] = depth
				froms[link] = from
				mu.Unlock()
			}

			if depth >= depth_target {
				continue
			}

			idxs = append(idxs, i)
		}

		for _, i := range idxs {
			time.Sleep(500 * time.Microsecond)
			e.Request.Visit(links[i])
		}
	})

	c.Visit(abs_source)
	c.Wait()

	var path []string

	if _, ok := froms[target]; !ok {
		return nil, errors.New("something went wrong :(")
	}
	for cur := target; cur != source; cur = froms[cur] {
		path = append(path, cur)
	}
	path = append(path, source)

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}
