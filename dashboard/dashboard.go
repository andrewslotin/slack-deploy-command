package dashboard

import (
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/andrewslotin/michael/deploy"
)

var dashboardTemplate = template.Must(
	template.New("dashboard").
		Funcs(template.FuncMap{
			"ftime": func(t time.Time) string { return t.Format(time.RFC822) },
		}).
		Parse(strings.TrimSpace(`
Deploy history
--------------

{{ range . -}}
{{ if not .FinishedAt.IsZero -}}
  * {{ .User.Name }} was deploying {{ .Subject }} since {{ .StartedAt | ftime }} until {{ .FinishedAt | ftime }}
{{ else -}}
  * {{ .User.Name }} is currently deploying {{ .Subject }} since {{ .StartedAt | ftime }}
{{ end -}}
{{ else -}}
  No deploys in channel so far
{{ end }}
`)))

type Dashboard struct {
	repo deploy.Repository
}

func New(repo deploy.Repository) *Dashboard {
	return &Dashboard{
		repo: repo,
	}
}

func (h *Dashboard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Path[1:]
	if channelID == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	dashboardTemplate.Execute(w, h.repo.All(channelID))
}

// ChannelIDFromRequest extracts and returns channelID from request URL.
func ChannelIDFromRequest(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return ""
	}

	n := strings.IndexByte(path, '/')
	if n < 0 {
		return path
	}

	return path[:n]
}