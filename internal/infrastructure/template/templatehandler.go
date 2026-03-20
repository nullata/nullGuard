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

// parses the base templates and stores them in baseTemplate
// called once at startup
func InitializeBaseTemplates() {
	baseTemplate = template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/partials/header.html",
		"templates/partials/footer.html",
		"templates/partials/modal.html",
		"templates/partials/tooltip.html",
	))
	log.Println("Base templates initialized")
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
		ErrorPageHandler(w, "Error cloning template", http.StatusInternalServerError)
		return
	}
	// render the template
	err = tmplClone.ExecuteTemplate(w, "base", data)
	if err != nil {
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
