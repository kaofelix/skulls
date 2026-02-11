package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/kaofelix/skulls/internal/skillsapi"
)

type SearchResult struct {
	Selected bool
	Skill    skillsapi.Skill
}

type SearchOptions struct {
	InitialSkills []skillsapi.Skill
	Placeholder   string
	StatusHint    string
	SearchFunc    func(context.Context, string, int) ([]skillsapi.Skill, error)
	PreviewFunc   func(context.Context, skillsapi.Skill) (string, error)
}

// RunSearch runs the interactive search UI in the alt screen and returns the selected skill.
func RunSearch() (SearchResult, error) {
	return RunSearchWithOptions(SearchOptions{})
}

func RunSearchWithOptions(opts SearchOptions) (SearchResult, error) {
	m := newSearchModelWithOptions(opts)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
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

type popularResultMsg struct {
	skills []skillsapi.Skill
	err    error
}

type triggerSearchMsg struct {
	seq   int
	query string
}

type previewResultMsg struct {
	seq int
	key string
	md  string
	err error
}

type searchModel struct {
	client skillsapi.Client

	input       textinput.Model
	results     list.Model
	allItems    []list.Item
	searchSeq   int
	searching   bool
	searchErr   error
	spinner     spinner.Model
	statusHint  string
	searchFunc  func(context.Context, string, int) ([]skillsapi.Skill, error)
	previewFunc func(context.Context, skillsapi.Skill) (string, error)

	popularLoading bool
	popularErr     error
	popularItems   []list.Item

	// Layout
	windowW      int
	windowH      int
	bodyH        int
	listW        int
	previewPaneW int

	// Preview
	previewSeq      int
	previewLoading  bool
	previewKey      string
	previewMarkdown string
	previewRendered string
	previewErr      error
	previewCache    map[string]string // key -> raw markdown
	previewVP       viewport.Model
	lastSelKey      string

	result SearchResult
}

const (
	fixedListWidth      = 48
	minPreviewPaneWidth = 30
)

func newSearchModel() searchModel {
	return newSearchModelWithOptions(SearchOptions{})
}

func newSearchModelWithOptions(opts SearchOptions) searchModel {
	ti := textinput.New()
	placeholder := strings.TrimSpace(opts.Placeholder)
	if placeholder == "" {
		placeholder = "Search skills…"
	}
	ti.Placeholder = placeholder
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

	allItems := make([]list.Item, 0, len(opts.InitialSkills))
	for _, sk := range opts.InitialSkills {
		allItems = append(allItems, skillItem{s: sk})
	}

	m := searchModel{
		client:         skillsapi.Client{},
		popularLoading: len(allItems) == 0,
		input:          ti,
		results:        l,
		allItems:       allItems,
		spinner:        s,
		statusHint:     strings.TrimSpace(opts.StatusHint),
		searchFunc:     opts.SearchFunc,
		previewFunc:    opts.PreviewFunc,
		previewCache:   map[string]string{},
		previewVP:      viewport.New(0, 0),
	}
	if len(allItems) > 0 {
		m.results.SetItems(allItems)
	}
	return m
}

func (m searchModel) Init() tea.Cmd {
	if len(m.allItems) > 0 {
		return tea.Batch(textinput.Blink, m.spinner.Tick, m.ensurePreviewForSelection())
	}
	return tea.Batch(textinput.Blink, m.spinner.Tick, doPopular(m.client, 50))
}

func (m searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowW = msg.Width
		m.windowH = msg.Height

		// Layout: input + blank line + status + blank line + body.
		m.bodyH = msg.Height - 4
		if m.bodyH < 1 {
			m.bodyH = 1
		}

		m.listW = fixedListWidth
		if msg.Width < fixedListWidth+minPreviewPaneWidth {
			m.listW = msg.Width
			m.previewPaneW = 0
		} else {
			if m.listW > msg.Width {
				m.listW = msg.Width
			}
			m.previewPaneW = msg.Width - m.listW
			if m.previewPaneW < 0 {
				m.previewPaneW = 0
			}
		}

		m.results.SetSize(m.listW, m.bodyH)

		m.previewVP.Width = wrapWidthForPreview(m.previewPaneW)
		// Reserve one line for a scroll indicator.
		previewH := m.bodyH - 1
		if previewH < 1 {
			previewH = 1
		}
		m.previewVP.Height = previewH

		// If we already have preview markdown, re-render for the new width.
		m.rerenderPreview()

		return m, nil

	case tea.MouseMsg:
		oldSelKey := m.selectedKey()

		// If scrolling in the preview pane, scroll preview instead of the list.
		if m.isInPreviewPane(msg) && msg.Action == tea.MouseActionPress && tea.MouseEvent(msg).IsWheel() {
			var cmd tea.Cmd
			m.previewVP, cmd = m.previewVP.Update(msg)
			return m, cmd
		}

		// Otherwise let the list handle the mouse event (wheel changes selection, clicks, etc.).
		var listCmd tea.Cmd
		m.results, listCmd = m.results.Update(msg)
		newSelKey := m.selectedKey()
		if newSelKey != "" && newSelKey != oldSelKey {
			return m, tea.Batch(listCmd, m.ensurePreviewForSelection())
		}
		return m, listCmd

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

		// Preview scrolling (keep list navigation separate).
		if m.previewPaneW > 0 && m.previewErr == nil && !m.previewLoading && strings.TrimSpace(m.previewRendered) != "" {
			switch msg.String() {
			case "pgdown":
				m.previewVP.PageDown()
				return m, nil
			case "pgup":
				m.previewVP.PageUp()
				return m, nil
			case "ctrl+d":
				m.previewVP.HalfPageDown()
				return m, nil
			case "ctrl+u":
				m.previewVP.HalfPageUp()
				return m, nil
			case "home":
				m.previewVP.GotoTop()
				return m, nil
			case "end":
				m.previewVP.GotoBottom()
				return m, nil
			}
		}

		oldQuery := m.input.Value()
		oldSelKey := m.selectedKey()

		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)

		var listCmd tea.Cmd
		m.results, listCmd = m.results.Update(msg)

		// Selection changed (up/down) -> update preview.
		newSelKey := m.selectedKey()
		var previewCmd tea.Cmd
		if newSelKey != "" && newSelKey != oldSelKey {
			previewCmd = m.ensurePreviewForSelection()
		}

		if m.input.Value() != oldQuery {
			q := strings.TrimSpace(m.input.Value())
			m.searchErr = nil

			if len(m.allItems) > 0 {
				m.searching = false
				if q == "" {
					m.results.SetItems(m.allItems)
				} else {
					filtered := make([]list.Item, 0, len(m.allItems))
					needle := strings.ToLower(q)
					for _, it := range m.allItems {
						si, ok := it.(skillItem)
						if !ok {
							continue
						}
						if strings.Contains(strings.ToLower(si.s.SkillID), needle) {
							filtered = append(filtered, it)
						}
					}
					m.results.SetItems(filtered)
				}
				previewCmd = m.ensurePreviewForSelection()
				return m, tea.Batch(inputCmd, listCmd, previewCmd)
			}

			// If query changed, debounce a search.
			m.searchSeq++
			if q == "" {
				m.searching = false
				// Show popular-by-default.
				if m.popularItems != nil {
					m.results.SetItems(m.popularItems)
					previewCmd = m.ensurePreviewForSelection()
				}
				if !m.popularLoading && m.popularItems == nil {
					m.popularLoading = true
					return m, tea.Batch(inputCmd, listCmd, previewCmd, doPopular(m.client, 50))
				}
				return m, tea.Batch(inputCmd, listCmd, previewCmd)
			}

			if len([]rune(q)) < 2 {
				m.searching = false
				m.results.SetItems([]list.Item{})
				m.clearPreview()
				return m, tea.Batch(inputCmd, listCmd)
			}
			seq := m.searchSeq
			return m, tea.Batch(inputCmd, listCmd, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
				return triggerSearchMsg{seq: seq, query: q}
			}))
		}

		return m, tea.Batch(inputCmd, listCmd, previewCmd)

	case triggerSearchMsg:
		if msg.seq != m.searchSeq {
			return m, nil
		}
		m.searching = true
		return m, doSearch(m.client, m.searchFunc, msg.query, 10, msg.seq)

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
		return m, tea.Batch(m.ensurePreviewForSelection())

	case popularResultMsg:
		m.popularLoading = false
		m.popularErr = msg.err
		items := make([]list.Item, 0, len(msg.skills))
		for _, s := range msg.skills {
			items = append(items, skillItem{s: s})
		}
		m.popularItems = items
		if strings.TrimSpace(m.input.Value()) == "" {
			m.results.SetItems(m.popularItems)
			return m, tea.Batch(m.ensurePreviewForSelection())
		}
		return m, nil

	case previewResultMsg:
		if msg.seq != m.previewSeq {
			return m, nil
		}
		m.previewLoading = false
		m.previewErr = msg.err
		m.previewKey = msg.key
		m.previewMarkdown = msg.md
		if msg.err == nil {
			m.previewCache[msg.key] = msg.md
		}
		m.previewVP.GotoTop()
		m.rerenderPreview()
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
	var status string

	if len(m.allItems) > 0 {
		status = m.statusHint
		if status == "" {
			status = "Filter skills • Enter to install • Esc to quit"
		}
	} else {
		switch {
		case q == "":
			switch {
			case m.popularLoading:
				status = m.spinner.View() + " Popular…"
			case m.popularErr != nil:
				status = "Error loading popular: " + m.popularErr.Error()
			default:
				status = "Popular • Type to search • Enter to install • Esc to quit"
			}
		case len([]rune(q)) < 2:
			status = "Type at least 2 characters to search."
		default:
			switch {
			case m.searching:
				status = m.spinner.View() + " Searching…"
			case m.searchErr != nil:
				status = "Error: " + m.searchErr.Error()
			default:
				status = "Enter to install • Esc to quit"
			}
		}
	}

	body := m.bodyView()

	// Intentionally no trailing newline: if we exceed terminal height by one line,
	// Bubble Tea will clip the top, which can hide the input line.
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.input.View(),
		status,
		body,
	)
}

func (m searchModel) isInPreviewPane(msg tea.MouseMsg) bool {
	if m.previewPaneW <= 0 {
		return false
	}
	// Layout: input + blank line + status + blank line take 4 rows.
	if msg.Y < 4 {
		return false
	}
	// List is fixed width from the left.
	return msg.X >= m.listW
}

func (m searchModel) bodyView() string {
	left := lipgloss.NewStyle().Width(m.listW).MaxWidth(m.listW).Render(m.results.View())
	if m.previewPaneW <= 0 {
		return left
	}

	right := m.previewView()

	// Create a small gap via padding on the right pane.
	rightStyled := lipgloss.NewStyle().
		Width(m.previewPaneW).
		MaxWidth(m.previewPaneW).
		MaxHeight(m.bodyH).
		PaddingLeft(2).
		Render(right)

	leftStyled := lipgloss.NewStyle().
		Width(m.listW).
		MaxWidth(m.listW).
		MaxHeight(m.bodyH).
		Render(m.results.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)
}

func (m searchModel) previewIndicatorLine() string {
	w := m.previewVP.Width
	if w <= 0 {
		return ""
	}

	// Only show indicators if the content can actually scroll.
	scrollable := m.previewVP.TotalLineCount() > m.previewVP.Height
	if !scrollable {
		return lipgloss.NewStyle().Width(w).Render("")
	}

	up := " "
	if !m.previewVP.AtTop() {
		up = "▲"
	}
	down := " "
	if !m.previewVP.AtBottom() {
		down = "▼"
	}
	pct := int(m.previewVP.ScrollPercent()*100 + 0.5)
	core := fmt.Sprintf("%s %3d%% %s", up, pct, down)

	// Build a centered bar like: ──── ▲  12% ▼ ────
	core = " " + core + " "
	fill := w - lipgloss.Width(core)
	if fill < 0 {
		fill = 0
	}
	left := fill / 2
	right := fill - left
	line := strings.Repeat("─", left) + core + strings.Repeat("─", right)

	return lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Foreground(lipgloss.AdaptiveColor{Light: "#4a6a88", Dark: "#8aa4c8"}).
		Render(line)
}

func (m searchModel) previewView() string {
	if m.selectedKey() == "" {
		return ""
	}

	if m.previewLoading {
		return m.spinner.View() + " Loading preview…"
	}

	if m.previewErr != nil {
		if errors.Is(m.previewErr, skillsapi.ErrPreviewUnavailable) {
			return "Preview unavailable"
		}
		return "Preview unavailable: " + m.previewErr.Error()
	}

	if strings.TrimSpace(m.previewRendered) == "" {
		return "Preview unavailable"
	}

	indicator := m.previewIndicatorLine()
	previewBlock := m.previewVP.View() + "\n" + indicator

	// Center the preview block in the available preview pane width.
	availableW := m.previewPaneW - 2 // match right pane padding
	if availableW < 0 {
		availableW = 0
	}
	previewBlock = lipgloss.PlaceHorizontal(availableW, lipgloss.Center, previewBlock)
	return previewBlock
}

func (m *searchModel) clearPreview() {
	m.previewLoading = false
	m.previewKey = ""
	m.previewMarkdown = ""
	m.previewRendered = ""
	m.previewErr = nil
	m.previewVP.GotoTop()
	m.previewVP.SetContent("")
	m.lastSelKey = ""
}

func (m *searchModel) rerenderPreview() {
	if m.previewPaneW <= 0 {
		m.previewRendered = ""
		m.previewVP.SetContent("")
		return
	}
	if m.previewMarkdown == "" {
		return
	}
	wrap := wrapWidthForPreview(m.previewPaneW)
	rendered, err := renderMarkdownANSI(m.previewMarkdown, wrap)
	if err != nil {
		// Fall back to raw markdown if rendering fails.
		m.previewRendered = m.previewMarkdown
		m.previewVP.SetContent(m.previewRendered)
		return
	}
	m.previewRendered = rendered
	m.previewVP.SetContent(m.previewRendered)
}

func (m *searchModel) selectedSkill() (skillsapi.Skill, bool) {
	it, ok := m.results.SelectedItem().(skillItem)
	if !ok {
		return skillsapi.Skill{}, false
	}
	return it.s, true
}

func (m *searchModel) selectedKey() string {
	s, ok := m.selectedSkill()
	if !ok {
		return ""
	}
	return previewKeyForSkill(s)
}

func previewKeyForSkill(s skillsapi.Skill) string {
	return strings.TrimSpace(s.Source) + "|" + strings.TrimSpace(s.SkillID)
}

func (m *searchModel) ensurePreviewForSelection() tea.Cmd {
	s, ok := m.selectedSkill()
	if !ok {
		m.clearPreview()
		return nil
	}

	key := previewKeyForSkill(s)
	m.lastSelKey = key

	if md, ok := m.previewCache[key]; ok {
		m.previewLoading = false
		m.previewErr = nil
		m.previewKey = key
		m.previewMarkdown = md
		m.previewVP.GotoTop()
		m.rerenderPreview()
		return nil
	}

	m.previewSeq++
	seq := m.previewSeq
	m.previewLoading = true
	m.previewErr = nil
	m.previewKey = key
	m.previewMarkdown = ""
	m.previewRendered = ""
	m.previewVP.GotoTop()
	m.previewVP.SetContent("")
	return doPreview(m.client, m.previewFunc, s, key, seq)
}

func renderMarkdownANSI(md string, wrap int) (string, error) {
	// Avoid glamour's auto style here: it calls termenv.HasDarkBackground(), which
	// queries the terminal and can cause escape sequence responses to land in the
	// Bubble Tea text input.
	style := strings.TrimSpace(os.Getenv("GLAMOUR_STYLE"))
	if style == "" || strings.EqualFold(style, "auto") {
		style = "dark"
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return "", err
	}

	rendered, err := r.Render(md)
	if err != nil {
		return "", err
	}
	// Glamour sometimes leads with newlines; drop them so content is top-aligned.
	rendered = strings.TrimLeft(rendered, "\n")
	return rendered, nil
}

func doSearch(client skillsapi.Client, searchFn func(context.Context, string, int) ([]skillsapi.Skill, error), query string, limit int, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		fn := searchFn
		if fn == nil {
			fn = client.Search
		}
		skills, err := fn(ctx, query, limit)
		return searchResultMsg{seq: seq, skills: skills, err: err}
	}
}

func doPopular(client skillsapi.Client, limit int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		skills, err := client.Popular(ctx, limit)
		return popularResultMsg{skills: skills, err: err}
	}
}

func doPreview(client skillsapi.Client, previewFn func(context.Context, skillsapi.Skill) (string, error), skill skillsapi.Skill, key string, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		fn := previewFn
		if fn == nil {
			fn = client.FetchSkillMarkdown
		}
		md, err := fn(ctx, skill)
		return previewResultMsg{seq: seq, key: key, md: md, err: err}
	}
}
