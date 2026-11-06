package workflow

import (
	"embed"
	"text/template"

	"github.com/alexfalkowski/abgleich/internal/markdown"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/strings"
)

//go:embed *.tmpl
var reviewFiles embed.FS

var reviewTemplates = template.Must(template.New("review").Funcs(template.FuncMap{
	"markdown": markdown.Inert,
	"lower":    strings.ToLower,
}).ParseFS(reviewFiles, "*.tmpl"))

func render(name string, value any) (string, error) {
	var body strings.Builder
	if err := reviewTemplates.ExecuteTemplate(&body, name, value); err != nil {
		return strings.Empty, errors.Prefix("review: render "+name, err)
	}
	return body.String(), nil
}
