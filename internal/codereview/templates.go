package codereview

import (
	"embed"
	"text/template"

	"github.com/alexfalkowski/go-service/v2/bytes"
	"github.com/alexfalkowski/go-service/v2/strings"
)

//go:embed *.tmpl
var reviewFiles embed.FS

//go:embed prompt.txt
var instructions string

var reviewTemplates = template.Must(template.New("codereview").Funcs(template.FuncMap{
	"markdown": markdownText,
}).ParseFS(reviewFiles, "*.tmpl"))

func render(name string, value any) (string, error) {
	var body bytes.Buffer
	if err := reviewTemplates.ExecuteTemplate(&body, name, value); err != nil {
		return strings.Empty, err
	}

	return body.String(), nil
}
