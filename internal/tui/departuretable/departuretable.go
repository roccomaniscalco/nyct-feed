package departuretable

import (
	"nyct-feed/internal/gtfs"
	"nyct-feed/internal/tui/routebadge"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var style = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240"))

type Model struct {
	table table.Model
}

func (m *Model) SetDepartures(departures []gtfs.Departure) {
	rows := []table.Row{}
	for _, d := range departures {
		departureTime := time.Unix(d.Times[0], 0).Format("15:04:05")
		routeBadge := routebadge.Render(d.Route)
		r := table.Row{routeBadge, d.FinalStopName, departureTime}
		rows = append(rows, r)
	}
	m.table.SetRows(rows)
}

func (m *Model) SetHeight(height int) {
	m.table.SetHeight(height - 3)
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	return style.Render(m.table.View()) + "\n"
}

func NewModel() Model {
	columns := []table.Column{
		{Title: "Route", Width: 5},
		{Title: "Destination", Width: 40},
		{Title: "Departs In", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(s)

	return Model{t}
}

// Route: 1                                                                                                           
// Raw badge: "\x1b[48;2;216;34;51m \x1b[0m\x1b[1;38;2;255;255;255;48;2;216;34;51m1\x1b[0m\x1b[48;2;216;34;51m \x1b[0m"
// String length: 85
// Visual width: 3
