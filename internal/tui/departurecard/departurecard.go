package departurecard

import (
	"fmt"
	"math"
	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/tui/routebadge"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var width = 60

var (
	strong   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}
	subtle   = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	border   = lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"}
	realtime = lipgloss.AdaptiveColor{Light: "#23854c", Dark: "#00dd8c"}
)

type Model struct {
	height     int
	station    gtfs.Station
	departures []gtfs.Departure
}

func NewModel() Model {
	return Model{}
}

func (m *Model) SetHeight(height int) {
	m.height = height - 2 // Top and bottom border
}

func (m *Model) SetStation(station gtfs.Station) {
	m.station = station
}

func (m *Model) SetDepartures(departures []gtfs.Departure) {
	m.departures = departures
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

var baseStyle = lipgloss.NewStyle().
	Width(width).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(border)

var titleStyle = lipgloss.NewStyle().
	Width(width).
	Padding(0, 1).
	Foreground(strong).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(border)

var mutedTextStyle = lipgloss.NewStyle().
	Foreground(subtle)

var routeHeadingStyle = lipgloss.NewStyle().
	Width(width).
	Padding(1, 1, 0, 1).
	BorderForeground(border)

var departureRowStyle = lipgloss.NewStyle().
	Width(width).
	Padding(0, 1).
	Foreground(strong)
var departureInnerWidth = departureRowStyle.GetWidth() - departureRowStyle.GetHorizontalFrameSize()

var (
	directionStyle   = lipgloss.NewStyle().PaddingRight(1).Foreground(strong)
	destinationStyle = lipgloss.NewStyle().Foreground(strong)
	timesStyle       = lipgloss.NewStyle().PaddingRight(1).Foreground(strong)
	realtimeStyle    = lipgloss.NewStyle().Foreground(realtime)
	spacingStyle     = lipgloss.NewStyle().Foreground(border)
)

var w = lipgloss.Width

func (m *Model) View() string {
	now := time.Now()
	content := []string{}

	title := titleStyle.Render(m.station.StopName)
	content = append(content, title)

	for _, route := range m.station.Routes {
		badge := routebadge.Render(route)
		longName := mutedTextStyle.Render(route.RouteLongName)
		heading := routeHeadingStyle.Render(badge + " " + longName)
		content = append(content, heading)

		for _, departure := range m.departures {
			if departure.RouteId == route.RouteId {
				direction := directionStyle.Render("(" + string(departure.StopId[len(departure.StopId)-1]) + ")")
				destination := destinationStyle.Render(departure.FinalStopName)
				timesStr := timesStyle.Render(getFormattedDepartureTimes(departure.Times, now))
				realtime := realtimeStyle.Render("•")
				availableWidth := departureInnerWidth - w(direction) - w(destination) - w(timesStr) - w(realtime)
				spacing := spacingStyle.Render(strings.Repeat(" ", max(1,availableWidth)))

				departureRow := departureRowStyle.Render(lipgloss.JoinHorizontal(
					lipgloss.Left,
					direction,
					destination,
					spacing,
					timesStr,
					realtime,
				))

				content = append(content, departureRow)
			}
		}
	}

	return baseStyle.Height(m.height).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			content...,
		),
	)
}

func getFormattedDepartureTimes(departureTimes []int64, now time.Time) string {
	slices.Sort(departureTimes)

	durations := []string{}
	for _, t := range departureTimes {
		if len(durations) == 2 {
			break
		}
		minTilDeparture := math.Round(time.Unix(t, 0).Sub(now).Minutes())
		if minTilDeparture > 0 {
			durations = append(durations, fmt.Sprintf("%v", minTilDeparture))
		} else if minTilDeparture == 0 {
			durations = append(durations, "Now")
		}
	}

	return fmt.Sprintf("%s min", strings.Join(durations, ", "))
}
