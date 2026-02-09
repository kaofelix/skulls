package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kaofelix/skulls/internal/skillsapi"
)

type SearchResult struct {
	Selected bool
	Skill    skillsapi.Skill
}

// RunSearch runs the interactive search UI in the alt screen and returns the selected skill.
func RunSearch() (SearchResult, error) {
	m := newSearchModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return SearchResult{}, err
	}
	fm, ok := finalModel.(searchModel)
	if !ok {
		return SearchResult{}, fmt.Errorf("unexpected model type %T", finalModel)
	}
	return fm.result, nil
}

type skillItem struct{ s skillsapi.Skill }

func (i skillItem) Title() string { return i.s.SkillID }
func (i skillItem) Description() string {
	parts := []string{}
	if i.s.Source != "" {
		parts = append(parts, i.s.Source)
	}
	if i.s.Installs > 0 {
		parts = append(parts, fmt.Sprintf("%d installs", i.s.Installs))
	}
	return strings.Join(parts, " • ")
}
func (i skillItem) FilterValue() string { return i.s.SkillID }

type searchResultMsg struct {
	seq    int
	skills []skillsapi.Skill
	err    error
}

type triggerSearchMsg struct {
	seq   int
	query string
}

type searchModel struct {
	client skillsapi.Client

	input     textinput.Model
	results   list.Model
	searchSeq int
	searching bool
	searchErr error
	spinner   spinner.Model

	result SearchResult
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = "Search skills…"
	ti.Focus()
	ti.Prompt = "> "

	s := spinner.New()
	s.Spinner = spinner.Line

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.SetShowFilter(false)
	l.Title = ""

	return searchModel{
		client:  skillsapi.Client{},
		input:   ti,
		results: l,
		spinner: s,
	}
}

func (m searchModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Layout: input (1 line) + status (1 line) + results.
		height := msg.Height - 2
		if height < 1 {
			height = 1
		}
		m.results.SetSize(msg.Width, height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if it, ok := m.results.SelectedItem().(skillItem); ok {
				m.result = SearchResult{Selected: true, Skill: it.s}
				return m, tea.Quit
			}
		}

		oldQuery := m.input.Value()

		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)

		var listCmd tea.Cmd
		m.results, listCmd = m.results.Update(msg)

		if m.input.Value() != oldQuery {
			// If query changed, debounce a search.
			m.searchSeq++
			q := strings.TrimSpace(m.input.Value())
			m.searchErr = nil
			if len([]rune(q)) < 2 {
				m.searching = false
				m.results.SetItems([]list.Item{})
				return m, tea.Batch(inputCmd, listCmd)
			}
			seq := m.searchSeq
			return m, tea.Batch(inputCmd, listCmd, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
				return triggerSearchMsg{seq: seq, query: q}
			}))
		}

		return m, tea.Batch(inputCmd, listCmd)

	case triggerSearchMsg:
		if msg.seq != m.searchSeq {
			return m, nil
		}
		m.searching = true
		return m, doSearch(m.client, msg.query, 10, msg.seq)

	case searchResultMsg:
		if msg.seq != m.searchSeq {
			return m, nil
		}
		m.searching = false
		m.searchErr = msg.err
		items := make([]list.Item, 0, len(msg.skills))
		for _, s := range msg.skills {
			items = append(items, skillItem{s: s})
		}
		m.results.SetItems(items)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	}

	return m, nil
}

func (m searchModel) View() string {
	q := strings.TrimSpace(m.input.Value())
	status := ""
	if len([]rune(q)) > 0 && len([]rune(q)) < 2 {
		status = "Type at least 2 characters to search."
	}
	if m.searching {
		status = m.spinner.View() + " Searching…"
	}
	if m.searchErr != nil {
		status = "Error: " + m.searchErr.Error()
	}
	if status == "" {
		status = "Enter to install • Esc to quit"
	}

	// Intentionally no trailing newline: if we exceed terminal height by one line,
	// Bubble Tea will clip the top, which can hide the input line.
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.input.View(),
		status,
		m.results.View(),
	)
}

func doSearch(client skillsapi.Client, query string, limit int, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		skills, err := client.Search(ctx, query, limit)
		return searchResultMsg{seq: seq, skills: skills, err: err}
	}
}
