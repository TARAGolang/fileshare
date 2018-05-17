package main

import (
	"html/template"
	"io"

	"github.com/labstack/echo"
)

const indexhtml = `<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>Upload file</title>
</head>

<body style="font-family: monospace; font-size: 14px;">
    <form action="" method="post" enctype="multipart/form-data">
        <input type="file" name="file" size="32">
        <input type="submit" value="Upload">
    </form>
    {{range .}}
    <p>
    <a href="/newlink/gen?file={{.}}">{{.}}</a>
    </p>
    {{end}}
</body>

</html>
`

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
