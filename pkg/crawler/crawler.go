package crawler

import (
"context"
"fmt"
"os"
"sync"
"time"

"github.com/pkg/errors"

"Go-BstPr-GB/pkg/parser"

)

type crawlResult struct {
	err error
	msg string
}

type crawler struct {
	sync.Mutex
	visited     map[string]string
	maxDepth    int
	userSignal1 chan struct{}
}

func NewCrawler(maxDepth int, userSignal1 chan struct{}) *crawler {
	return &crawler{
		visited:     make(map[string]string),
		maxDepth:    maxDepth,
		userSignal1: userSignal1,
	}
}



// рекурсивно сканируем страницы
func (c *crawler) run(ctx context.Context, url string, results chan<- crawlResult, depth int) {
	// просто для того, чтобы успевать следить за выводом программы, можно убрать :)
	time.Sleep(5 * time.Second)

	config, err := CreateNew()
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

	// проверка глубины
	if depth >= c.maxDepth {
		return
	}

	// проверяем что контекст исполнения актуален
	select {
	case <-ctx.Done():
		return

	case <-c.userSignal1:
		c.maxDepth += 2

	default:
		page, err := parse(config.Url)
		if err != nil {
			// ошибку отправляем в канал, а не обрабатываем на месте
			results <- crawlResult{
				err: errors.Wrapf(err, "parse page %s", url),
			}
			return
		}

		title := pageTitle(page)
		links := pageLinks(nil, page)

		// блокировка требуется, т.к. мы модифицируем мапку в несколько горутин
		c.Lock()
		c.visited[url] = title
		c.Unlock()

		// отправляем результат в канал, не обрабатывая на месте
		results <- crawlResult{
			err: nil,
			msg: fmt.Sprintf("%s -> %s\n", url, title),
		}

		// рекурсивно ищем ссылки
		for link := range links {
			// если ссылка не найдена, то запускаем анализ по новой ссылке
			if c.checkVisited(link) {
				continue
			}

			go c.run(ctx, link, results, depth+1)
		}
	}
}

func (c *crawler) checkVisited(url string) bool {
	c.Lock()
	defer c.Unlock()

	_, ok := c.visited[url]
	return ok
}