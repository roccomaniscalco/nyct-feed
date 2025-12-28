package departurecards

import (
	"nyct-feed/internal/gtfs"

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

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Model) View() string {
	title := titleStyle.Render(m.station.StopName)
	return baseStyle.Height(m.height).Render(title)
}
