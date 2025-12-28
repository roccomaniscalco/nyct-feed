package stationlist

import (
	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/tui/routebadge"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const width = 40

type StationSelectedMsg string

type item struct {
	station     gtfs.Station
	routeBadges string
}

func (i item) Title() string       { return i.station.StopName }
func (i item) Description() string { return i.routeBadges }
func (i item) FilterValue() string { return i.station.StopName }

type Model struct {
	selectedStationId string
	list              list.Model
}

func NewModel() Model {
	list := list.New([]list.Item{}, list.NewDefaultDelegate(), width, 0)
	list.SetWidth(width)

	list.SetShowPagination(false)
	list.SetShowHelp(false)
	list.SetShowStatusBar(false)
	list.DisableQuitKeybindings()

	list.Styles.TitleBar = lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		MarginBottom(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})

	list.Title = renderKbd("/") + titleStyle.Render("Search Stations")
	list.Styles.Title = titleStyle

	list.FilterInput.Prompt = renderKbd("/")
	list.FilterInput.CharLimit = width - list.Styles.TitleBar.GetHorizontalFrameSize() - 1
	list.FilterInput.Placeholder = "Search Stations"
	list.FilterInput.TextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

	return Model{
		list: list,
	}
}

func (m *Model) SetHeight(height int) {
	m.list.SetHeight(height)
}

func (m *Model) SetStations(stations []gtfs.Station) {
	items := []list.Item{}
	for _, station := range stations {
		routeBadges := strings.Builder{}
		for _, route := range station.Routes {
			routeBadges.WriteString(routebadge.Render(route))
			routeBadges.WriteString(" ")
		}
		items = append(items, item{station: station, routeBadges: routeBadges.String()})
	}
	m.list.SetItems(items)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	updatedList, listCmd := m.list.Update(msg)
	m.list = updatedList
	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}

	// Emit station selection change
	if !m.list.SettingFilter() && m.list.SelectedItem() != nil {
		if item, ok := m.list.SelectedItem().(item); ok {
			if item.station.StopId != m.selectedStationId {
				m.selectedStationId = item.station.StopId
				stationSelectedCmd := func() tea.Msg {
					return StationSelectedMsg(item.station.StopId)
				}
				cmds = append(cmds, stationSelectedCmd)
			}
		}
	}

	if m.list.SettingFilter() {
		m.list.Styles.TitleBar = m.list.Styles.TitleBar.
			BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"})
	} else {
		m.list.Styles.TitleBar = m.list.Styles.TitleBar.
			BorderForeground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})
	}

	if m.list.FilterValue() == "" {
		m.list.Title = renderKbd("/") + titleStyle.Render("Search Stations")
	} else {
		m.list.Title = renderKbd("/") + titleStyle.Render(m.list.FilterValue())
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return m.list.View()
}

func renderKbd(key string) string {
	style := lipgloss.NewStyle().
		MarginRight(1).
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

	return style.Render(key)
}

var titleStyle = lipgloss.NewStyle().
	UnsetBackground().
	Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})
