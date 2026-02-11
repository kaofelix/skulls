package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

type InstallResult struct {
	InstalledPath string
	Err           error
}

// RunInstall shows an install progress UI in the *normal* terminal screen (no alt screen).
// It exits when the install is complete, leaving the final checklist visible in scrollback.
func RunInstall(targetDir string, force bool, skill skillsapi.Skill) (InstallResult, error) {
	m := newInstallModel(targetDir, force, skill)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return InstallResult{}, err
	}
	fm, ok := finalModel.(installModel)
	if !ok {
		return InstallResult{}, fmt.Errorf("unexpected model type %T", finalModel)
	}
	return InstallResult{InstalledPath: fm.installedPath, Err: fm.err}, nil
}

type installEventMsg install.Event

type installDoneMsg struct {
	path string
	err  error
}

type installModel struct {
	targetDir string
	force     bool
	skill     skillsapi.Skill

	spin spinner.Model

	steps map[install.Step]install.Event
	order []install.Step

	installedPath string
	err           error

	msgCh <-chan tea.Msg
}

func newInstallModel(targetDir string, force bool, skill skillsapi.Skill) installModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := installModel{
		targetDir: targetDir,
		force:     force,
		skill:     skill,
		spin:      s,
		steps:     map[install.Step]install.Event{},
		order: []install.Step{
			install.StepNormalize,
			install.StepClone,
			install.StepVerify,
			install.StepCopy,
		},
	}
	m.msgCh = startInstall(m.targetDir, m.force, m.skill)
	return m
}

func (m installModel) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, waitMsg(m.msgCh))
}

func (m installModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = tea.ErrProgramKilled
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case installEventMsg:
		e := install.Event(msg)
		m.steps[e.Step] = e
		// If we see StepRemove, include it in the order (it only appears when needed).
		if e.Step == install.StepRemove && !containsStep(m.order, install.StepRemove) {
			// Insert before copy.
			newOrder := []install.Step{}
			for _, s := range m.order {
				if s == install.StepCopy {
					newOrder = append(newOrder, install.StepRemove)
				}
				newOrder = append(newOrder, s)
			}
			m.order = newOrder
		}
		return m, waitMsg(m.msgCh)
	case installDoneMsg:
		m.installedPath = msg.path
		m.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func (m installModel) View() string {
	headerStyle := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ok := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	pending := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	bad := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	b := strings.Builder{}
	b.WriteString(headerStyle.Render("Installing " + m.skill.SkillID))
	if m.skill.Source != "" {
		b.WriteString(muted.Render("  (" + m.skill.Source + ")"))
	}
	b.WriteString("\n\n")

	const lineWidth = 52
	for _, step := range m.order {
		e, seen := m.steps[step]
		label := stepLabel(step)

		if !seen {
			b.WriteString(muted.Render(formatStepLine(label, "·", lineWidth)) + "\n")
			continue
		}

		if e.Done {
			b.WriteString(ok.Render(formatStepLine(label, "✓", lineWidth)) + "\n")
			continue
		}
		// Current step.
		b.WriteString(pending.Render(formatStepLine(label, m.spin.View(), lineWidth)) + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + bad.Render("✗ "+m.err.Error()) + "\n")
	}

	return b.String()
}

func formatStepLine(label string, status string, width int) string {
	label = strings.TrimSpace(label)
	status = strings.TrimSpace(status)
	if label == "" {
		label = "Step"
	}
	if status == "" {
		status = "·"
	}
	if width < 20 {
		width = 20
	}

	minDots := 3
	dots := width - lipgloss.Width(label) - lipgloss.Width(status) - 2
	if dots < minDots {
		dots = minDots
	}
	return label + " " + strings.Repeat(".", dots) + " " + status
}

func startInstall(targetDir string, force bool, skill skillsapi.Skill) <-chan tea.Msg {
	ch := make(chan tea.Msg, 128)
	go func() {
		path, err := install.InstallSkill(skill.Source, skill.SkillID, install.Options{
			TargetDir: targetDir,
			Force:     force,
			GitStdout: io.Discard,
			GitStderr: io.Discard,
			Progress: func(e install.Event) {
				ch <- installEventMsg(e)
			},
		})
		ch <- installDoneMsg{path: path, err: err}
		close(ch)
	}()
	return ch
}

func stepLabel(step install.Step) string {
	switch step {
	case install.StepNormalize:
		return "Normalize source"
	case install.StepClone:
		return "Download repository"
	case install.StepVerify:
		return "Verify skill layout"
	case install.StepRemove:
		return "Remove existing installation"
	case install.StepCopy:
		return "Copy files"
	default:
		return string(step)
	}
}

func containsStep(steps []install.Step, s install.Step) bool {
	for _, x := range steps {
		if x == s {
			return true
		}
	}
	return false
}

func waitMsg(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
