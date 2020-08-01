package utils

import (
	"fmt"
	"github.com/AngryYogurt/ToolBox/mixamo/config"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func BuildHeader(r *http.Request) {
	for k, v := range config.Headers {
		if k == "Cookie" {
			u := time.Now().UnixNano() / int64(time.Millisecond)
			v += fmt.Sprintf(config.CookieTail, u, u)
		}
		r.Header.Add(k, v)
	}
}

func Request(client *http.Client, r *http.Request) []byte {
	var err error
	var resp *http.Response
	resp, err = client.Do(r)
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted) {
		log.Println("http.go:27", err, resp)
	}
	respData, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Println("http.go:32", err)
	}
	return respData
}
