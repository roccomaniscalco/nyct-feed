package departurecard

import (
	"fmt"
	"math"
	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/tui/routebadge"
	"nyct-feed/internal/tui/theme"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var width = 60

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
	BorderForeground(theme.Border)

var titleStyle = lipgloss.NewStyle().
	Width(width).
	Padding(0, 1).
	Foreground(theme.Strong).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(theme.Border)

var mutedTextStyle = lipgloss.NewStyle().
	Foreground(theme.Subtle)

var routeHeadingStyle = lipgloss.NewStyle().
	Width(width).
	Padding(1, 1, 0, 1).
	BorderForeground(theme.Border)

var departureRowStyle = lipgloss.NewStyle().
	Width(width).
	Padding(0, 1).
	Foreground(theme.Strong)
var departureInnerWidth = departureRowStyle.GetWidth() - departureRowStyle.GetHorizontalFrameSize()

var (
	directionStyle   = lipgloss.NewStyle().PaddingRight(1).Foreground(theme.Strong)
	destinationStyle = lipgloss.NewStyle().Foreground(theme.Strong)
	timesStyle       = lipgloss.NewStyle().PaddingRight(1).Foreground(theme.Strong)
	realtimeStyle    = lipgloss.NewStyle().Foreground(theme.Realtime)
	spacingStyle     = lipgloss.NewStyle().Foreground(theme.Border)
)

var w = lipgloss.Width

func (m *Model) View() string {
	now := time.Now()
	content := []string{}

	title := titleStyle.Render(m.station.StopName)
	content = append(content, title)

	for _, route := range m.station.Routes {
		badge := routebadge.RenderOne(route)
		longName := mutedTextStyle.Render(route.RouteLongName)
		heading := routeHeadingStyle.Render(badge + " " + longName)
		content = append(content, heading)

		for _, departure := range m.departures {
			if departure.RouteId == route.RouteId {
				departureTimes := getFormattedDepartureTimes(departure.Times, now)
				if departureTimes == "No Departures" {
					continue
				}

				timesStr := timesStyle.Render(departureTimes)
				direction := directionStyle.Render("(" + string(departure.StopId[len(departure.StopId)-1]) + ")")
				destination := destinationStyle.Render(departure.FinalStopName)
				realtime := realtimeStyle.Render("•")
				availableWidth := departureInnerWidth - w(direction) - w(destination) - w(timesStr) - w(realtime)
				spacing := spacingStyle.Render(strings.Repeat(" ", max(1, availableWidth)))

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

// getFormattedDepartureTimes returns the soonest upcoming departures as a string.
// If there are no upcoming departures "No Departures" is returned.
// Example: "Now, 8 min"
func getFormattedDepartureTimes(departureTimes []int64, now time.Time) string {
	slices.Sort(departureTimes)

	durations := []string{}
	for _, t := range departureTimes {
		if len(durations) == 3 {
			break
		}
		minTilDeparture := math.Round(time.Unix(t, 0).Sub(now).Minutes())
		if minTilDeparture > 0 {
			durations = append(durations, fmt.Sprintf("%v", minTilDeparture))
		} else if minTilDeparture == 0 {
			durations = append(durations, "Now")
		}
	}

	if len(durations) == 0 {
		return "No Departures"
	} 

	suffix := ""
	if durations[len(durations) -1] != "Now" {
		suffix = " min"
	}
	return fmt.Sprintf("%s%s", strings.Join(durations, ", "), suffix)
}
