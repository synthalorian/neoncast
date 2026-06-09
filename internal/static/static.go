package static

import (
	"embed"
	"io/fs"
)

//go:embed dashboard.html
var content embed.FS

// FS returns the embedded filesystem for static assets.
func FS() fs.FS {
	return content
}

// DashboardHTML returns the embedded dashboard.html file.
func DashboardHTML() (fs.File, error) {
	return content.Open("dashboard.html")
}

// ReadDashboardHTML returns the contents of dashboard.html as bytes.
func ReadDashboardHTML() ([]byte, error) {
	return content.ReadFile("dashboard.html")
}
