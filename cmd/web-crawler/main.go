package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// максимально допустимое число ошибок при парсинге
	errorsLimit = 100000

	// число результатов, которые хотим получить
	resultsLimit = 10000
)

type Config struct {
	Url string `json:"url"`
}

var (
	// насколько глубоко нам надо смотреть (например, 10)
	depthLimit int
	jsonFile   = flag.String("jsonFile", "tsconfig.json", "add link to json config file")
)

// Как вы помните, функция инициализации стартует первой
func init() {
	// задаём и парсим флаги
	flag.IntVar(&depthLimit, "depth", 3, "max depth for run")
	flag.Parse()
}

func CreateNew() (*Config, error) {

	var config *Config

	if *jsonFile == "" {
		config = &Config{
			"",
		}
	} else {
		confJsonFile, err := ioutil.ReadFile(*jsonFile)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(confJsonFile, &config)
		if err != nil {
			log.Println(err)
		}

	}

	_, err := url.ParseRequestURI(config.Url)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func main() {

	config, err := CreateNew()
	if err != nil {
		fmt.Println("error", err)
	}
	if config.Url == "" {
		log.Print("no url set by flag")
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Println(os.Getpid())

	started := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	userSignal1 := make(chan struct{})

	go watchSignals(cancel)

	go UserSig(userSignal1)

	defer func() {
		close(userSignal1)
	}()

	defer cancel()

	crawler := NewCrawler(depthLimit, userSignal1)

	// создаём канал для результатов
	results := make(chan crawlResult)

	// запускаем горутину для чтения из каналов
	done := watchCrawler(ctx, results, errorsLimit, resultsLimit)

	// запуск основной логики
	// внутри есть рекурсивные запуски анализа в других горутинах
	crawler.run(ctx, config.Url, results, 0)

	// ждём завершения работы чтения в своей горутине
	<-done

	log.Println(time.Since(started))

}

func UserSig(UserSignal1 chan struct{}) {
	USig := make(chan os.Signal, 1)
	signal.Notify(USig,
		syscall.SIGUSR1)
	fmt.Println(<-USig)
	UserSignal1 <- struct{}{}
}

// ловим сигналы выключения
func watchSignals(cancel context.CancelFunc) {
	osSignalChan := make(chan os.Signal, 2)

	signal.Notify(osSignalChan,
		syscall.SIGINT,
		syscall.SIGTERM)

	sig := <-osSignalChan
	log.Printf("got signal %q", sig.String())

	// если сигнал получен, отменяем контекст работы
	cancel()
}

func watchCrawler(ctx context.Context, results <-chan crawlResult, maxErrors, maxResults int) chan struct{} {
	readersDone := make(chan struct{})

	go func() {
		defer close(readersDone)
		for {
			select {
			case <-ctx.Done():
				return

			case result := <-results:
				if result.err != nil {
					maxErrors--
					if maxErrors <= 0 {
						log.Println("max errors exceeded")
						return
					}
					continue
				}

				log.Printf("crawling result: %v", result.msg)
				maxResults--
				if maxResults <= 0 {
					log.Println("got max results")
					return
				}
			}
		}
	}()

	return readersDone
}