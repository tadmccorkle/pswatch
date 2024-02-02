package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type Proc struct {
	Pid         int
	Exe         string
	Description string
	Path        string
	Parent      int
	User        string
}

type model struct {
	procs  []Proc
	cursor int
	height int
	min    int
	max    int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height - 6
		m.max = m.height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.min {
					m.min--
					m.max--
				}
			}
		case "down", "j":
			if m.cursor < len(m.procs) {
				m.cursor++
				if m.cursor > m.max {
					m.min++
					m.max++
				}
			}
			// case "enter", " ":
			// 	_, ok := m.selected[m.cursor]
			// 	if ok {
			// 		delete(m.selected, m.cursor)
			// 	} else {
			// 		m.selected[m.cursor] = struct{}{}
			// }
		}
	}

	return m, nil
}

func (m model) View() string {
	s := "Processes\n\n"

	for i, proc := range m.procs {
		if i < m.min {
			continue
		}
		if i > m.max {
			break
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		name := proc.Exe
		if proc.Description != "" {
			name = proc.Description + " (" + name + ")"
		}

		s += fmt.Sprintf("%s [%6d] %-50s (%s)\n", cursor, proc.Pid, name, proc.User)
	}

	s += "\nPress q to quit.\n"

	return s
}

func main() {
	procs := getProcs()

	p := tea.NewProgram(model{procs: procs})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
