package rendering

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

type Renderer interface {
	Render(templateName string, data any, w io.Writer, r *http.Request)
	RenderRaw(templateString string, data any, w io.Writer, r *http.Request)
}

type TemplateRendererConfig struct {
	TemplateFS fs.FS
}

type TemplateRenderer struct {
	templateFS fs.FS
}

func NewTemplateRenderer(config TemplateRendererConfig) TemplateRenderer {
	return TemplateRenderer{
		templateFS: config.TemplateFS,
	}
}

func (tr TemplateRenderer) getFuncs() template.FuncMap {
	templateFuncs := template.FuncMap{
		"join":                strings.Join,
		"isSet":               templateFuncIsSet,
		"isLastItem":          tr.isLastItem,
		"containsString":      containsString,
		"stringSliceContains": sliceContains[string],
		"uintSliceContains":   sliceContains[uint],
	}

	return templateFuncs
}

func (tr TemplateRenderer) Render(templateName string, data any, w io.Writer, r *http.Request) {
	var (
		err  error
		tmpl *template.Template
	)

	templateFuncs := tr.getFuncs()
	templates := []string{
		"templates/" + templateName + ".tmpl",
		"templates/layout.tmpl",
	}

	if tmpl, err = template.New(templateName+".tmpl").Funcs(templateFuncs).ParseFS(tr.templateFS, templates...); err != nil {
		slog.Error("error parsing template", slog.Any("error", err))
		fmt.Fprintf(w, "error parsing template: %s", err.Error())
		return
	}

	if err = tmpl.Execute(w, data); err != nil {
		slog.Error("error executing template", slog.Any("error", err))
		fmt.Fprintf(w, "error executing template: %s", err.Error())
	}
}

func (tr TemplateRenderer) RenderRaw(templateString string, data any, w io.Writer, r *http.Request) {
	var (
		err  error
		tmpl *template.Template
	)

	templateFuncs := tr.getFuncs()

	if tmpl, err = template.New("raw").Funcs(templateFuncs).Parse(templateString); err != nil {
		slog.Error("error parsing template", slog.Any("error", err))
		fmt.Fprintf(w, "error parsing template: %s", err.Error())
		return
	}

	if err = tmpl.Execute(w, data); err != nil {
		slog.Error("error executing template", slog.Any("error", err))
		fmt.Fprintf(w, "error executing template: %s", err.Error())
	}
}

func templateFuncIsSet(name string, data any) bool {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	return v.FieldByName(name).IsValid()
}

func sliceContains[T comparable](array []T, value T) bool {
	return slices.Index(array, value) > -1
}

func (tr TemplateRenderer) isLastItem(index, length int) bool {
	return index == length-1
}

func containsString(slice []string, item string) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}

	return false
}
