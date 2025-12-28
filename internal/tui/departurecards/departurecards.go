package departurecards

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
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"}).
	Width(40)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
	Bold(true).
	Width(40).
	Padding(0, 1).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})

var mutedTextStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

var routeHeadingStyle = lipgloss.NewStyle().
	Width(40).
	Padding(0, 1).
	MarginTop(1).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})

var departureItemStyle = lipgloss.NewStyle().
	Width(40).
	Padding(0, 1).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

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
				minutes := getMinuteTilDepartures(departure.Times, now)
				minutesStr := strings.Join(minutes, " ")

				departureItem := departureItemStyle.Render(departure.FinalStopName + " " + minutesStr)
				content = append(content, departureItem)
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

func getMinuteTilDepartures(departureTimes []int64, now time.Time) []string {
	// TODO: Redundant Sort
	slices.Sort(departureTimes)

	durations := []string{}
	for _, t := range departureTimes {
		if len(durations) == 2 {
			break
		}
		timeTilDeparture := time.Unix(t, 0).Sub(now)
		if timeTilDeparture > 0 {
			minTilDeparture := math.Round(timeTilDeparture.Minutes())
			durations = append(durations, fmt.Sprintf("%v", minTilDeparture))
		}
	}

	return durations
}
