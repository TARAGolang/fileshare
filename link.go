package main

import (
	"net/http"
	"net/url"
)

func createLink(c *http.Request, fnb, key string) string {
	u := &url.URL{}
	*u = *c.URL
	u.Path = "/" + fnb
	u.Host = c.Host
	u.Scheme = "https"
	uv := url.Values{"key": []string{key}}
	u.RawQuery = uv.Encode()
	return u.String()
}
