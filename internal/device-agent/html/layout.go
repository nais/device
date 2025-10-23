package html

import (
	"embed"
	"html/template"
	"io"
	"strings"
	"time"
)

//go:embed layout.html
var layout embed.FS

var funcs = template.FuncMap{
	"formatTime": func(t time.Time) string {
		return t.Format("02/01/2006, 15:04:05")
	},
	"nl2br": func(s string) template.HTML {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		escaped := template.HTMLEscapeString(s)
		escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
		return template.HTML(escaped)
	},
}

func Render(w io.Writer, templates embed.FS, path string, data any) error {
	t := template.New("layout.html").Funcs(funcs)
	t, err := t.ParseFS(layout, "layout.html")
	if err != nil {
		return err
	}

	t, err = t.ParseFS(templates, path)
	if err != nil {
		return err
	}

	return t.ExecuteTemplate(w, "layout.html", data)
}
