package splash

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var asciiLogo = `
                       __   
     ____  __  _______/ /_  
    / __ \/ / / / ___/ __/  
   / / / / /_/ / /__/ /_    
  /_/ /_/\__, /\___/\___)   
        (____/              
                            
`

type Model struct{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m Model) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#2e63c5"))
	return style.Render(asciiLogo)
}
