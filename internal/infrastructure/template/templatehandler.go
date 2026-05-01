// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package template

import (
	"html/template"
	"log"
	"net/http"

	"nullguard/internal/api/http/middleware"
)

var baseTemplate *template.Template

// baseFuncs are template helper functions available to every parsed
// template. Keep this small; the broader the surface, the more places that
// need to think about template rendering vs. data shaping.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		// mkCheat builds the data payload for tooltip-with-cheatsheet so the
		// template can address fields by name. Avoids needing a full dict
		// helper just for this one partial.
		"mkCheat": func(text, key, title string) map[string]string {
			return map[string]string{"Text": text, "Key": key, "Title": title}
		},
	}
}

// parses the base templates and stores them in baseTemplate
// called once at startup
func InitializeBaseTemplates() {
	baseTemplate = template.Must(template.New("base.html").Funcs(baseFuncs()).ParseFiles(
		"templates/base.html",
		"templates/partials/header.html",
		"templates/partials/footer.html",
		"templates/partials/modal.html",
		"templates/partials/tooltip.html",
	))
	// cheatsheets/*.html each define a {{define "cheatsheet-<key>"}} block
	// served by the help handler. ParseGlob will fail if the directory is
	// empty; ship the placeholder set so the glob always matches.
	baseTemplate = template.Must(baseTemplate.ParseGlob("templates/cheatsheets/*.html"))
	log.Println("Base templates initialized")
}

// LookupTemplate reports whether a named template exists on the base
// template. Used by handlers that render named partials directly (e.g.
// cheatsheet bodies returned by /help/{key}).
func LookupTemplate(name string) bool {
	return baseTemplate != nil && baseTemplate.Lookup(name) != nil
}

// RenderTemplate executes a named template and writes the output to w.
// Always renders on a clone  html/template forbids Clone() after a
// template has been executed, and baseTemplate must stay pristine so the
// per-request page rendering pipeline (cloneBaseTemplate) keeps working.
// Caller is responsible for setting Content-Type before invoking this.
func RenderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	clone, err := baseTemplate.Clone()
	if err != nil {
		return err
	}
	return clone.ExecuteTemplate(w, name, data)
}

// returns a clone of the base template with additional files parsed
func cloneBaseTemplate(files ...string) (*template.Template, error) {
	tmplClone, err := baseTemplate.Clone()
	if err != nil {
		return nil, err
	}

	// parse additional files into the cloned template
	tmplClone, err = tmplClone.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return tmplClone, nil
}

func TemplateHandler(w http.ResponseWriter, template string, data interface{}) {
	// inject session max age into template data for client-side auto-logout
	if m, ok := data.(map[string]interface{}); ok {
		m["SessionMaxAge"] = middleware.GetSessionMaxAge()
	}

	// clone the template for rendering the page
	tmplClone, err := cloneBaseTemplate(template)
	if err != nil {
		log.Printf("Error cloning template %q: %v", template, err)
		ErrorPageHandler(w, "Error cloning template", http.StatusInternalServerError)
		return
	}
	// render the template
	err = tmplClone.ExecuteTemplate(w, "base", data)
	if err != nil {
		log.Printf("Error executing template %q: %v", template, err)
		ErrorPageHandler(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

// GetTemplate returns a parsed template for standalone pages (like setup and login)
func GetTemplate(filename string) *template.Template {
	tmpl, err := template.ParseFiles(
		"templates/"+filename,
		"templates/partials/modal.html",
	)
	if err != nil {
		log.Printf("Error parsing template %s: %v", filename, err)
		return nil
	}
	return tmpl
}
