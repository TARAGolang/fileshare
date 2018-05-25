package handlers

import (
	"net/http"
	"net/url"
)

func createLink(c *http.Request, fnb, key string) string {
	u := &url.URL{}
	*u = *c.URL
	return CreateLinkFromURL(u, c.Host, fnb, key)
}

func CreateLinkFromURL(u *url.URL, host, fnb, key string) string {
	u.Path = "/" + fnb
	u.Host = host
	u.Scheme = "https"
	uv := url.Values{"key": []string{key}}
	u.RawQuery = uv.Encode()
	return u.String()
}
