package main

import (
	"log"
	"regexp"
	"strconv"
	"sync"

	"github.com/gocolly/colly"
)

var sellers map[string]int

func scrapSearchResults(url string, productsLinks *chan string) {
	if url == "" {
		log.Println("missing url argument")
		return
	}

	log.Println("visiting", url)

	c := colly.NewCollector()
	c.OnHTML(".andes-pagination__button--next", func(e *colly.HTMLElement) {
		go scrapSearchResults(e.ChildAttr(".andes-pagination__link", "href"), productsLinks)
	})
	c.Visit(url)

	c1 := colly.NewCollector()
	c1.OnHTML(".ui-search-result__content-wrapper", func(e *colly.HTMLElement) {
		*productsLinks <- e.ChildAttr("a", "href")
	})
	c1.Visit(url)

}

func getProductInfo(url string, wg *sync.WaitGroup, m *sync.Mutex) {

	if url == "" {
		log.Println("missing url argument")
		return
	}

	c := colly.NewCollector()
	c.OnHTML(".layout-col--right", func(e *colly.HTMLElement) {
		re := regexp.MustCompile(`\d+`)
		itemCount, _ := strconv.Atoi(string(re.Find([]byte(e.ChildText(".item-conditions")))))
		id := e.ChildAttr("#seller-view-more-link", "href")

		m.Lock()
		count, exists := sellers[id]
		if exists {
			sellers[id] = count + itemCount
		} else {
			sellers[id] = itemCount
		}
		m.Unlock()

		wg.Done()
	})
	c.Visit(url)
}

func scrapProductPages(productsLinks *chan string, wg *sync.WaitGroup, m *sync.Mutex) {

	for {
		go func() {
			select {
			case url := <-*productsLinks:
				wg.Add(1)
				go getProductInfo(url, wg, m)
			default:
			}
		}()
	}

}

func main() {
	// https://celulares.mercadolibre.com.co/_Desde_451_BestSellers_YES
	var wg sync.WaitGroup
	var m sync.Mutex
	productsLinks := make(chan string)
	go scrapSearchResults("https://celulares.mercadolibre.com.co/_BestSellers_YES", &productsLinks)
	go scrapProductPages(&productsLinks, &wg, &m)
	wg.Wait()
	log.Println(sellers)
}
