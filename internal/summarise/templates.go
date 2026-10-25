package summarise

import (
	"embed"
	"text/template"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/strings"
)

//go:embed *.tmpl
var summaryFiles embed.FS

//go:embed prompt.txt
var instructions string

var summaryTemplates = template.Must(template.New("summarise").Funcs(template.FuncMap{
	"markdown": markdownText,
}).ParseFS(summaryFiles, "*.tmpl"))

func render(name string, value any) (string, error) {
	var body bytes.Buffer
	if err := summaryTemplates.ExecuteTemplate(&body, name, value); err != nil {
		return strings.Empty, err
	}

	return body.String(), nil
}
