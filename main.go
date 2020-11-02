package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gocolly/colly"
)

var sellers map[string]int
var m sync.Mutex
var countP int64

var wg sync.WaitGroup
var wgP sync.WaitGroup

func scrapSearchResults(url string, searchLinks *chan string, productsLinks *chan string) {
	if url == "" {
		log.Println("missing url argument")
		return
	}

	fmt.Println("visiting", url)

	cls := true
	c := colly.NewCollector()

	c.OnHTML(".andes-pagination__button--next", func(e *colly.HTMLElement) {
		link := e.ChildAttr(".andes-pagination__link", "href")
		*searchLinks <- link
		cls = false
	})
	err := c.Visit(url)
	if err != nil {
		fmt.Println("err", err)
	}
	if cls {
		close(*searchLinks)
		wg.Done()
		return
	}

	c1 := colly.NewCollector()
	c1.OnHTML(".ui-search-result__content-wrapper", func(e *colly.HTMLElement) {
		*productsLinks <- e.ChildAttr("a", "href")
	})
	c1.Visit(url)

}

func scrapProduct(url string) {
	atomic.AddInt64(&countP, 1)
	defer wgP.Done()

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

func initScrapSearchResults(searchLinks *chan string, productsLinks *chan string) {
	for url := range *searchLinks {
		go scrapSearchResults(url, searchLinks, productsLinks)
	}
}

func initScrapProduct(productsLinks *chan string) {
	for url := range *productsLinks {
		wgP.Add(1)
		go scrapProduct(url)

	}
}

func main() {
	fmt.Println("let's start")

	productsLinks := make(chan string, 10)
	searchLinks := make(chan string, 10)
	sellers = make(map[string]int)
	countP = 0

	// searchLinks <- "https://celulares.mercadolibre.com.co/_BestSellers_YES"
	searchLinks <- "https://celulares.mercadolibre.com.co/_Desde_1901_BestSellers_YES"

	wg.Add(1)

	go initScrapSearchResults(&searchLinks, &productsLinks)
	go initScrapProduct(&productsLinks)

	wg.Wait()
	wgP.Wait()

	log.Println(sellers)
	fmt.Println(countP)
}
