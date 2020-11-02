package main

import (
	"log"
	"regexp"
	"strconv"
	"sync"

	"github.com/gocolly/colly"
)

var sellers map[string]int
var m sync.Mutex

func scrapSearchResults(url string, searchLinks, productsLinks *chan string) {
	if url == "" {
		log.Println("missing url argument")
		return
	}

	c := colly.NewCollector()
	c.OnHTML(".andes-pagination__button--next", func(e *colly.HTMLElement) {
		*searchLinks <- e.ChildAttr(".andes-pagination__link", "href")
	})
	c.Visit(url)

	c1 := colly.NewCollector()
	c1.OnHTML(".ui-search-result__content-wrapper", func(e *colly.HTMLElement) {
		*productsLinks <- e.ChildAttr("a", "href")
	})
	c1.Visit(url)

}

func scrapProduct(url string) {

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
	})
	c.Visit(url)
}

func initScrapSearchResults(searchLinks, productsLinks *chan string) {
	for url := range *searchLinks {
		go scrapSearchResults(url, searchLinks, productsLinks)
	}
}

func initScrapProduct(productsLinks *chan string) {
	for url := range *productsLinks {
		go scrapProduct(url)
	}
}

func main() {
	productsLinks := make(chan string)
	searchLinks := make(chan string)
	searchLinks <- "https://celulares.mercadolibre.com.co/_BestSellers_YES"
	go initScrapSearchResults(&searchLinks, &productsLinks)
	go initScrapProduct(&productsLinks)
	log.Println(sellers)
}
