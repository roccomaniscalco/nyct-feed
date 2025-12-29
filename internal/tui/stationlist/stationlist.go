package stationlist

import (
	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/tui/routebadge"
	"nyct-feed/internal/tui/theme"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const width = 40

type StationSelectedMsg *gtfs.Station

type stationItem struct {
	gtfs.Station
}

func (i stationItem) Title() string       { return i.StopName }
func (i stationItem) Description() string { return routebadge.RenderMany(i.Routes) }
func (i stationItem) FilterValue() string { return i.StopName }

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
		BorderForeground(theme.Border)

	list.Title = renderKbd("/") + titleStyle.Render("Search Stations")
	list.Styles.Title = titleStyle

	list.FilterInput.Prompt = renderKbd("/")
	list.FilterInput.CharLimit = width - list.Styles.TitleBar.GetHorizontalFrameSize() - 1
	list.FilterInput.Placeholder = "Search Stations"
	list.FilterInput.TextStyle = lipgloss.NewStyle().
		Foreground(theme.Strong)

	return Model{
		list: list,
	}
}

func (m *Model) SetHeight(height int) {
	m.list.SetHeight(height)
}

func (m *Model) SetStations(stations []gtfs.Station) {
	stationItems := make([]list.Item, len(stations))
	for i, station := range stations {
		stationItems[i] = stationItem{station}
	}

	// Manually set list state to how it was before updating items
	// TODO: Causes filter cursor to stop blinking
	filterText := m.list.FilterValue()
	filterState := m.list.FilterState()
	index := m.list.Index()

	m.list.SetItems(stationItems)
	m.list.SetFilterText(filterText)
	m.list.SetFilterState(filterState)
	m.list.Select(index)
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
		if item, ok := m.list.SelectedItem().(stationItem); ok {
			if item.StopId != m.selectedStationId {
				m.selectedStationId = item.StopId
				stationSelectedCmd := func() tea.Msg {
					return StationSelectedMsg(&item.Station)
				}
				cmds = append(cmds, stationSelectedCmd)
			}
		}
	}

	if m.list.SettingFilter() {
		m.list.Styles.TitleBar = m.list.Styles.TitleBar.
			BorderForeground(theme.Active)
	} else {
		m.list.Styles.TitleBar = m.list.Styles.TitleBar.
			BorderForeground(theme.Border)
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
		Foreground(theme.Strong)

	return style.Render(key)
}

var titleStyle = lipgloss.NewStyle().
	UnsetBackground().
	Foreground(theme.Subtle)
