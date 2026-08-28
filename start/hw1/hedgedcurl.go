package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("ошибка формирования запроса к %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("сетевая ошибка при запросе к %s: %w", url, err)
	}

	defer resp.Body.Close()

	var result strings.Builder

	result.WriteString(fmt.Sprintf("%s %s\n", resp.Proto, resp.Status))

	for key, values := range resp.Header {
		for _, value := range values {
			result.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	result.WriteString("\n")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения данных от %s: %w", url, err)
	}

	result.Write(bodyBytes)

	return result.String(), nil
}

func main() {
	var timeout int

	flag.IntVar(&timeout, "t", 15, "Timeout in seconds")
	flag.IntVar(&timeout, "timeout", 15, "Timeout in seconds")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hedgedcurl [options] <url1> <url2> ...\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	urls := flag.Args()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	
	resultCh := make(chan string)
	errorCh := make(chan error)

	for _, u := range urls {
		go func(url string) {
			res, err := fetchURL(ctx, url)
			if err != nil {
				errorCh <- err
				return
			}
			
			select {
			case resultCh <- res: 
			case <-ctx.Done():    
			}
		}(u) 
	}

	errorsCount := 0
	for {
		select {
		case res := <-resultCh:
			fmt.Print(res)
			cancel() 
			return  
			

		case <-errorCh:
			errorsCount++

			if errorsCount == len(urls) {
				fmt.Fprintln(os.Stderr, "Ошибка: все запросы завершились неудачно")
				os.Exit(1)
			}
			
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Таймаут: ни один сервер не ответил за %d сек\n", timeout)
			os.Exit(228) 
		}
	}

}
