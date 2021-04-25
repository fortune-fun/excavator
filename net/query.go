package net

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

//由Unicode获取汉程cache
func CacheUnicode(base_url string, wd string) (*html.Node, error) {
	reader, e := cache.ReaderUnicode(base_url, wd)
	if e != nil {
		return nil, e
	}
	defer reader.Close()

	node, err := htmlquery.Parse(reader)

	return node, err
}

//由汉程字的URL获取cache
func CacheQuery(req_url string) (*html.Node, error) {
	reader, e := cache.Reader(req_url)
	if e != nil {
		return nil, e
	}
	defer reader.Close()

	node, err := htmlquery.Parse(reader)

	return node, err
}

//由Unicode获取url信息
//返回的页面不一定含有对应unicode的链接
// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
func UnicodeRequest(req_url string, wd string) (io.ReadCloser, error) {

	data := url.Values{
		"Tid": {"10"},
		"wd":  {wd},
	}
	dataStr := data.Encode()

	client := &http.Client{}

	req, err := http.NewRequest("POST", req_url, strings.NewReader(dataStr))
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (;) / (,) / / ")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	req.Header.Set("Content-Length", strconv.Itoa(len(dataStr)))

	resp, err := client.Do(req)
	if err != nil {
		Log.Panic(err)
	}

	if resp.StatusCode != 200 {
		Log.Fatal("Http status code:", resp.StatusCode)
	}
	//defer resp.Body.Close()
	//return htmlquery.Parse(resp.Body)
	return resp.Body, nil
}

//由字的url获取页面
func GetRequest(req_url string) (io.ReadCloser, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", req_url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (;) / (,) / / ")

	resp, err := client.Do(req)
	if err != nil {
		Log.Panic(err)
	}

	if resp.StatusCode != 200 {
		Log.Fatal("Http status code:", resp.StatusCode)
	}
	//defer resp.Body.Close()
	//return htmlquery.Parse(resp.Body)
	return resp.Body, nil
}
