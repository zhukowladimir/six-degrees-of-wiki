package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
)

var mu sync.Mutex

func main() {
	abs_source := "https://en.wikipedia.org/wiki/Corrosion_inhibitor"
	abs_target := "https://en.wikipedia.org/wiki/Wi-Fi"

	_, source, _ := strings.Cut(abs_source, "wikipedia.org")
	_, target, _ := strings.Cut(abs_target, "wikipedia.org")

	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(20),
		// colly.Debugger(&debug.LogDebugger{}),
		colly.AllowURLRevisit(),
	)
	extensions.RandomUserAgent(c)
	c.Limit(&colly.LimitRule{
		DomainRegexp: "https://[a-z]+\\.wikipedia\\.org/*.",
		Parallelism:  2,
		RandomDelay:  5 * time.Second,
	})
	// // errCount := make(map[colly.Request]uint)
	// c.OnError(func(r *colly.Response, err error) {
	// 	fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	// 	// if c, ok := errCount[*r.Request]; r.StatusCode != 200 && ok && c < 3 {
	// 	// 	if !ok {
	// 	// 		c = 0
	// 	// 	}
	// 	// 	errCount[*r.Request] = c + 1
	// 	// 	time.Sleep(5 * time.Second)
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
			e.Request.Visit(links[i])
		}
	})

	timer := time.Now()
	c.Visit(abs_source)
	c.Wait()

	fmt.Println("time: ", time.Since(timer).Seconds())
	fmt.Println("-------------")
	fmt.Println(depths[target])
	if _, ok := froms[target]; !ok {
		fmt.Println("gg :(")
	} else {
		for cur := target; cur != source; cur = froms[cur] {
			fmt.Print(cur, " <- ")
		}
		fmt.Println(source)
	}
	fmt.Println("-------------")
}
