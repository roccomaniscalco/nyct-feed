package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/pb"
	"nyct-feed/internal/query"
	"nyct-feed/internal/tui/departurecards"
	"nyct-feed/internal/tui/splash"
	"nyct-feed/internal/tui/stationlist"
)

type model struct {
	scheduleChannel chan query.Query[*gtfs.Schedule]
	realtimeChannel chan query.Query[[]*pb.FeedMessage]
	scheduleQuery   query.Query[*gtfs.Schedule]
	realtimeQuery   query.Query[[]*pb.FeedMessage]
	stationList     stationlist.Model
	departureCards  departurecards.Model
	selectedStation *gtfs.Station
	width           int
	height          int
}

func NewModel() model {
	return model{
		scheduleChannel: make(chan query.Query[*gtfs.Schedule]),
		realtimeChannel: make(chan query.Query[[]*pb.FeedMessage]),
		stationList:     stationlist.NewModel(),
		departureCards:  departurecards.NewModel(),
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		createScheduleQuery(m.scheduleChannel),
		createRealtimeQuery(m.realtimeChannel),
		getScheduleQuery(m.scheduleChannel),
		getRealtimeQuery(m.realtimeChannel),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.stationList.SetHeight(m.height)
		m.departureCards.SetHeight(m.height)
		return m, nil

	case gotScheduleQueryMsg:
		m.scheduleQuery = query.Query[*gtfs.Schedule](msg)
		if m.scheduleQuery.Data != nil {
			m.selectedStation = &m.scheduleQuery.Data.Stations[0]
		}
		m.syncStationList()
		m.syncDepartureCards()
		return m, getScheduleQuery(m.scheduleChannel)

	case gotRealtimeQueryMsg:
		m.realtimeQuery = query.Query[[]*pb.FeedMessage](msg)
		m.syncDepartureCards()
		return m, getRealtimeQuery(m.realtimeChannel)

	case stationlist.StationSelectedMsg:
		m.selectedStation = msg
		m.syncDepartureCards()
		return m, nil
	}

	var cmds []tea.Cmd
	if m.scheduleQuery.Data != nil {
		updatedModel, cmd := m.stationList.Update(msg)
		m.stationList = *updatedModel.(*stationlist.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.scheduleQuery.Status == query.Pending || m.realtimeQuery.Status == query.Pending {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(splash.Model{}.View())
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, m.stationList.View(), m.departureCards.View())
}

func (m *model) syncDepartureCards() {
	if m.scheduleQuery.Data != nil && m.realtimeQuery.Data != nil {
		stationId := m.selectedStation.StopId
		stopIds := []string{stationId + "N", stationId + "S"}
		departures := gtfs.FindDepartures(stopIds, m.realtimeQuery.Data, m.scheduleQuery.Data)
		m.departureCards.SetDepartures(departures)
		m.departureCards.SetStation(*m.selectedStation)
	}
}

func (m *model) syncStationList() {
	if m.scheduleQuery.Data != nil {
		stations := m.scheduleQuery.Data.Stations
		m.stationList.SetStations(stations)
	}
}

type gotScheduleQueryMsg query.Query[*gtfs.Schedule]

func getScheduleQuery(scheduleChannel chan query.Query[*gtfs.Schedule]) tea.Cmd {
	return func() tea.Msg {
		return gotScheduleQueryMsg(<-scheduleChannel)
	}
}

type gotRealtimeQueryMsg query.Query[[]*pb.FeedMessage]

func getRealtimeQuery(realtimeChannel chan query.Query[[]*pb.FeedMessage]) tea.Cmd {
	return func() tea.Msg {
		return gotRealtimeQueryMsg(<-realtimeChannel)
	}
}

func createScheduleQuery(scheduleChannel chan query.Query[*gtfs.Schedule]) tea.Cmd {
	return func() tea.Msg {
		query.CreateQuery[*gtfs.Schedule](query.QueryOptions[*gtfs.Schedule]{
			QueryChannel:    scheduleChannel,
			QueryFn:         gtfs.GetSchedule,
			RefetchInterval: time.Hour,
		})
		return nil
	}
}

func createRealtimeQuery(realtimeChannel chan query.Query[[]*pb.FeedMessage]) tea.Cmd {
	return func() tea.Msg {
		query.CreateQuery[[]*pb.FeedMessage](query.QueryOptions[[]*pb.FeedMessage]{
			QueryChannel:    realtimeChannel,
			QueryFn:         gtfs.GetRealtime,
			RefetchInterval: time.Second * 10,
		})
		return nil
	}
}
