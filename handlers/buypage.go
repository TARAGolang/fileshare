package handlers

import (
	"net/http"
	"sort"

	"github.com/labstack/echo"
)

type PayDesc struct {
	Key   string
	Price string
	Desc  string
}

func BuyPage(fprices, descr map[string]string) func(c echo.Context) error {
	return func(c echo.Context) error {
		fds := make([]PayDesc, 0)
		useddescr := make(map[string]bool)
		for k, v := range fprices {
			if d, ok := descr[k]; ok {
				fds = append(fds, PayDesc{Key: k, Price: v, Desc: d})
				useddescr[k] = true
			} else {
				fds = append(fds, PayDesc{Key: k, Price: v})
			}
		}
		for k, v := range descr {
			if _, ok := useddescr[k]; !ok {
				fds = append(fds, PayDesc{Key: k, Price: "", Desc: v})
			}
		}
		sort.Slice(fds, func(i, j int) bool { return fds[i].Key < fds[j].Key })

		return c.Render(http.StatusOK, "payment.html", fds)
	}
}
