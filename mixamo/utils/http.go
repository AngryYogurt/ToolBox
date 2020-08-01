package utils

import (
	"fmt"
	"github.com/AngryYogurt/ToolBox/mixamo/config"
	"io/ioutil"
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

func Request(client *http.Client, r *http.Request, retry int) ([]byte, error) {
	var err error
	var resp *http.Response
	resp, err = client.Do(r)
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted) {
		if retry > 10 {
			return nil, err
		}
		time.Sleep(2 * time.Second)
		return Request(client, r, retry+1)
	}
	respData, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		if retry > 10 {
			return nil, err
		}
		time.Sleep(2 * time.Second)
		return Request(client, r, retry+1)
	}
	return respData, nil
}
