package handlers

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/covrom/fileshare/store"

	"github.com/labstack/echo"
)

func UploadHtml(pth string) func(c echo.Context) error {
	return func(c echo.Context) error {
		files, err := ioutil.ReadDir(pth)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		allf := make([]string, 0, len(files))
		for _, info := range files {
			if info.Mode().IsRegular() {
				allf = append(allf, info.Name())
			}
		}
		return c.Render(http.StatusOK, "upload.html", allf)
	}
}

func UploadFile(pth string, str *store.Store) func(c echo.Context) error {
	return func(c echo.Context) error {
		f, err := c.FormFile("file")
		if err != nil {
			return err
		}
		if len(f.Filename) > 0 {
			src, err := f.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			fnb := filepath.Base(f.Filename)
			fn := filepath.Join(pth, fnb)
			// Destination
			dst, err := os.Create(fn)
			if err != nil {
				return err
			}
			defer dst.Close()

			// Copy
			if _, err = io.Copy(dst, src); err != nil {
				return err
			}
			return c.String(http.StatusOK, createLink(c.Request(), fnb, str.Set(fn)))
		}
		return c.String(http.StatusBadRequest, "Bad request")
	}
}
