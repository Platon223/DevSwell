package httpapi

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed web/index.html
var landingPageHTML []byte

//go:embed web/signup.html
var signupPageHTML []byte

//go:embed web/login.html
var loginPageHTML []byte

//go:embed web/dashboard
var dashboardTemplatesFS embed.FS

//go:embed web/static
var staticFilesRaw embed.FS

var dashboardSections = []string{"home", "stack", "feed", "settings", "upgrade", "help", "support"}

var dashboardTemplates = buildDashboardTemplates()

func buildDashboardTemplates() map[string]*template.Template {
	m := make(map[string]*template.Template, len(dashboardSections))
	for _, name := range dashboardSections {
		m[name] = template.Must(template.ParseFS(dashboardTemplatesFS, "web/dashboard/layout.html", "web/dashboard/"+name+".html"))
	}
	return m
}

type dashboardPageData struct {
	Active          string
	HelpButtonLabel string // empty means no floating help button on this page
	HelpButtonHref  string
}

// helpButtonFor returns the floating help-button label/href for a given
// dashboard section: most pages link to Help, Help itself links to Support
// (a step further), and Support has no help button at all (nowhere further
// to send someone).
func helpButtonFor(section string) (label, href string) {
	switch section {
	case "help":
		return "Still stuck?", "/dashboard/support"
	case "support":
		return "", ""
	default:
		return "Need help?", "/dashboard/help"
	}
}

func landingPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(landingPageHTML)
}

func signupPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(signupPageHTML)
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(loginPageHTML)
}

func dashboardPageHandler(section string) http.HandlerFunc {
	tmpl := dashboardTemplates[section]
	label, href := helpButtonFor(section)
	data := dashboardPageData{Active: section, HelpButtonLabel: label, HelpButtonHref: href}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func staticFileServer() http.Handler {
	staticFS, err := fs.Sub(staticFilesRaw, "web/static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
}
