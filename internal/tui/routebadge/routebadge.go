package routebadge

import (
	"nyct-feed/internal/gtfs"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderOne(route gtfs.Route) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#"+route.RouteTextColor)).
		Background(lipgloss.Color("#"+route.RouteColor)).
		Bold(true).
		Padding(0, 1)

	return style.Render(route.RouteShortName)
}

func RenderMany(routes []gtfs.Route) string {
	routeBadges := strings.Builder{}
	for _, route := range routes {
		routeBadges.WriteString(RenderOne(route))
		routeBadges.WriteString(" ")
	}
	return routeBadges.String()
}
