package net

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/godcong/excavator/net"
)

// TestNewCache ...
func TestNewCache(t *testing.T) {
	cache := NewCache("./tmp")
	//url := "https://pics.javbus.com/cover/6qx9_b.jpg"
	url := "https://i.pinimg.com/originals/b4/fe/e5/b4fee5af13373edddccd78e6d0c815e3.jpg"

	t.Log(cache.Get(url))

	t.Log(cache.Save(url, "./save/image.jpg"))

}

func TestCache(t *testing.T) {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	body := strings.NewReader(`wd=` + url.QueryEscape("明"))
	req, err := http.NewRequest("POST", "http://hy.httpcn.com/bushou/zi/", body)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (iPad; CPU OS 11_0 like Mac OS X) AppleWebKit/604.1.34 (KHTML, like Gecko) Version/11.0 Mobile/15A5341f Safari/604.1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	response, err := net.Request(req)
	if err != nil {
		panic(err)
	}
	closer := response.Body

	if cache != nil {
		name := req.URL.String()
		closer, err = cache.Cache(response.Body, name)
	}

	fmt.Println(closer, err)
}
