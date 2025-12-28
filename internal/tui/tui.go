package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/pb"
	"nyct-feed/internal/tui/departuretable"
	"nyct-feed/internal/tui/splash"
	"nyct-feed/internal/tui/stationlist"
)

type model struct {
	schedule          *gtfs.Schedule
	scheduleLoading   bool
	stations          []gtfs.Stop
	selectedStationId string
	realtime          []*pb.FeedMessage
	realtimeLoading   bool
	departures        []gtfs.Departure
	stationList       stationlist.Model
	departureTable    departuretable.Model

	width  int
	height int
}

func NewModel() model {
	return model{
		scheduleLoading: true,
		realtimeLoading: true,
		departureTable:  departuretable.NewModel(),
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(getSchedule(), getRealtime())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.departureTable.SetHeight(m.height)
	case gotScheduleMsg:
		m.schedule = msg
		m.stations = m.schedule.GetStations()
		m.selectedStationId = m.stations[0].StopId
		m.stationList = stationlist.NewModel(m.stations, m.schedule.Routes)
		m.stationList.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.scheduleLoading = false
		m.syncDeparturesTable()
	case gotRealtimeMsg:
		m.realtime = msg
		m.realtimeLoading = false
		m.syncDeparturesTable()
	case stationlist.StationSelectedMsg:
		stationId := string(msg)
		m.selectedStationId = stationId
		m.syncDeparturesTable()
	}

	// Update both components and batch their commands
	var cmds []tea.Cmd

	if !m.scheduleLoading {
		updatedModel, cmd := m.stationList.Update(msg)
		m.stationList = *updatedModel.(*stationlist.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	if !m.realtimeLoading {
		updatedModel, cmd := m.departureTable.Update(msg)
		m.departureTable = *updatedModel.(*departuretable.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.scheduleLoading || m.realtimeLoading {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(splash.Model{}.View())
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, m.stationList.View(), m.departureTable.View())
}

func (m *model) syncDeparturesTable() {
	if !m.realtimeLoading && !m.scheduleLoading {
		stationId := m.selectedStationId
		stopIds := []string{stationId + "N", stationId + "S"}
		m.departures = gtfs.FindDepartures(stopIds, m.realtime, m.schedule)
		m.departureTable.SetDepartures(m.departures)
	}
}

type gotScheduleMsg *gtfs.Schedule

func getSchedule() tea.Cmd {
	return func() tea.Msg {
		schedule, _ := gtfs.GetSchedule()
		return gotScheduleMsg(schedule)
	}
}

type gotRealtimeMsg []*pb.FeedMessage

func getRealtime() tea.Cmd {
	return func() tea.Msg {
		feeds, _ := gtfs.FetchFeeds()
		return gotRealtimeMsg(feeds)
	}
}
