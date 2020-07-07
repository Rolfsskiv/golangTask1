package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
)

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	select {
	case h.sem <- struct{}{}:
		defer func() {
			<-h.sem
		}()
	default:
		msg := "Server does not accept more than 100 requests"
		http.Error(w, msg, http.StatusTooManyRequests)
		return
	}

	_, cancel := context.WithCancel(r.Context())
	go func() {
		<-r.Context().Done()
		cancel()
	}()

	var urls []string
	if err := json.NewDecoder(r.Body).Decode(&urls); err != nil {
		http.Error(w, "Json is not valid", http.StatusBadRequest)
		return
	}
	if len(urls) == 0 {
		msg := "Request is empty"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if len(urls) > MaxUrlsInQuery {
		msg := "The number of URLs is too large. The length of urls should not exceed 20"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	for i := range urls {
		if !validateURL(urls[i]) {
			msg := "Got invalid url: " + urls[i]
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
	}

	result, err := doAllWork(r.Context(), urls)
	if err != nil {
		msg := "An error occurred while executing the request: " + err.Error()
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(&result)
	if err != nil {
		msg := "An error occurred while executing the request: " + err.Error()
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(bytes); err != nil {
		// errors should be handled anyway
	}
}

func validateURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}

func doAllWork(requestCtx context.Context, urls []string) (map[string]string, error) {
	var wg sync.WaitGroup
	urlsChan := make(chan string, len(urls))
	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(requestCtx)
	defer cancel()

	for _, u := range urls {
		urlsChan <- u
	}
	close(urlsChan)

	dataChan := make(chan []string, len(urls))

	workerLen := MaxWorker
	if workerLen > len(urls) {
		workerLen = len(urls)
	}
	for i := 0; i < workerLen; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					u, ok := <-urlsChan
					if !ok {
						return
					}

					result, err := getInfo(ctx, u)
					if err != nil {
						select {
						case errChan <- err:
							cancel()
						default:
							// prevent lock if we have got more than one error
							// or use errChan with len of urls
						}
						return
					}

					dataChan <- []string{u, result}
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)
	close(dataChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	var output = map[string]string{}
	for data := range dataChan {
		output[data[0]] = data[1]
	}

	return output, nil
}

func getInfo(ctx context.Context, url string) (string, error) {
	cl := http.Client{
		Timeout: TimeoutGetUrl,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.New("Bad HTTP Response")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
