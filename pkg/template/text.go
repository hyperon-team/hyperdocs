package templateutil

import "text/template"

// CreateTextTemplate is a helper function for creation of text templates.
func CreateTextTemplate(parent *template.Template, name, src string) *template.Template {
	if parent != nil {
		return template.Must(parent.New(name).Parse(src))
	}
	return template.Must(template.New(name).Parse(src))
}
