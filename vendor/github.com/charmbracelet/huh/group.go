package huh

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh/internal/selector"
	"github.com/charmbracelet/lipgloss"
)

// Group is a collection of fields that are displayed together with a page of
// the form. While a group is displayed the form completer can switch between
// fields in the group.
//
// If any of the fields in a group have errors, the form will not be able to
// progress to the next group.
type Group struct {
	// collection of fields
	selector *selector.Selector[Field]

	// information
	title       string
	description string

	// navigation
	viewport viewport.Model

	// help
	showHelp bool
	help     help.Model

	// errors
	showErrors bool

	// group options
	width  int
	height int
	keymap *KeyMap
	hide   func() bool
	active bool
}

// NewGroup returns a new group with the given fields.
func NewGroup(fields ...Field) *Group {
	selector := selector.NewSelector(fields)
	group := &Group{
		selector:   selector,
		help:       help.New(),
		showHelp:   true,
		showErrors: true,
		active:     false,
	}

	height := group.fullHeight()
	v := viewport.New(80, height) //nolint:mnd
	group.viewport = v
	group.height = height

	return group
}

// Title sets the group's title.
func (g *Group) Title(title string) *Group {
	g.title = title
	return g
}

// Description sets the group's description.
func (g *Group) Description(description string) *Group {
	g.description = description
	return g
}

// WithShowHelp sets whether or not the group's help should be shown.
func (g *Group) WithShowHelp(show bool) *Group {
	g.showHelp = show
	return g
}

// WithShowErrors sets whether or not the group's errors should be shown.
func (g *Group) WithShowErrors(show bool) *Group {
	g.showErrors = show
	return g
}

// WithTheme sets the theme on a group.
func (g *Group) WithTheme(t *Theme) *Group {
	g.help.Styles = t.Help
	g.selector.Range(func(_ int, field Field) bool {
		field.WithTheme(t)
		return true
	})
	if g.height <= 0 {
		g.WithHeight(g.fullHeight())
	}
	return g
}

// WithKeyMap sets the keymap on a group.
func (g *Group) WithKeyMap(k *KeyMap) *Group {
	g.keymap = k
	g.selector.Range(func(_ int, field Field) bool {
		field.WithKeyMap(k)
		return true
	})
	return g
}

// WithWidth sets the width on a group.
func (g *Group) WithWidth(width int) *Group {
	g.width = width
	g.viewport.Width = width
	g.selector.Range(func(_ int, field Field) bool {
		field.WithWidth(width)
		return true
	})
	return g
}

// WithHeight sets the height on a group.
func (g *Group) WithHeight(height int) *Group {
	g.height = height
	g.viewport.Height = height
	g.selector.Range(func(_ int, field Field) bool {
		// A field height must not exceed the form height.
		if height-1 <= lipgloss.Height(field.View()) {
			field.WithHeight(height)
		}
		return true
	})
	return g
}

// WithHide sets whether this group should be skipped.
func (g *Group) WithHide(hide bool) *Group {
	g.WithHideFunc(func() bool { return hide })
	return g
}

// WithHideFunc sets the function that checks if this group should be skipped.
func (g *Group) WithHideFunc(hideFunc func() bool) *Group {
	g.hide = hideFunc
	return g
}

// Errors returns the groups' fields' errors.
func (g *Group) Errors() []error {
	var errs []error
	g.selector.Range(func(_ int, field Field) bool {
		if err := field.Error(); err != nil {
			errs = append(errs, err)
		}
		return true
	})
	return errs
}

// updateFieldMsg is a message to update the fields of a group that is currently
// displayed.
//
// This is used to update all TitleFunc, DescriptionFunc, and ...Func update
// methods to make all fields dynamically update based on user input.
type updateFieldMsg struct{}

// nextFieldMsg is a message to move to the next field,
//
// each field controls when to send this message such that it is able to use
// different key bindings or events to trigger group progression.
type nextFieldMsg struct{}

// prevFieldMsg is a message to move to the previous field.
//
// each field controls when to send this message such that it is able to use
// different key bindings or events to trigger group progression.
type prevFieldMsg struct{}

// NextField is the command to move to the next field.
func NextField() tea.Msg {
	return nextFieldMsg{}
}

// PrevField is the command to move to the previous field.
func PrevField() tea.Msg {
	return prevFieldMsg{}
}

// Init initializes the group.
func (g *Group) Init() tea.Cmd {
	var cmds []tea.Cmd

	if g.selector.Selected().Skip() {
		if g.selector.OnLast() {
			cmds = append(cmds, g.prevField()...)
		} else if g.selector.OnFirst() {
			cmds = append(cmds, g.nextField()...)
		}
		return tea.Batch(cmds...)
	}

	if g.active {
		cmd := g.selector.Selected().Focus()
		cmds = append(cmds, cmd)
	}
	g.buildView()
	return tea.Batch(cmds...)
}

// nextField moves to the next field.
func (g *Group) nextField() []tea.Cmd {
	blurCmd := g.selector.Selected().Blur()
	if g.selector.OnLast() {
		return []tea.Cmd{blurCmd, nextGroup}
	}
	g.selector.Next()
	for g.selector.Selected().Skip() {
		if g.selector.OnLast() {
			return []tea.Cmd{blurCmd, nextGroup}
		}
		g.selector.Next()
	}
	focusCmd := g.selector.Selected().Focus()
	return []tea.Cmd{blurCmd, focusCmd}
}

// prevField moves to the previous field.
func (g *Group) prevField() []tea.Cmd {
	blurCmd := g.selector.Selected().Blur()
	if g.selector.OnFirst() {
		return []tea.Cmd{blurCmd, prevGroup}
	}
	g.selector.Prev()
	for g.selector.Selected().Skip() {
		if g.selector.OnFirst() {
			return []tea.Cmd{blurCmd, prevGroup}
		}
		g.selector.Prev()
	}
	focusCmd := g.selector.Selected().Focus()
	return []tea.Cmd{blurCmd, focusCmd}
}

// Update updates the group.
func (g *Group) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update all the fields in the group.
	g.selector.Range(func(i int, field Field) bool {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			break
		default:
			m, cmd := field.Update(msg)
			g.selector.Set(i, m.(Field))
			cmds = append(cmds, cmd)
		}
		if g.selector.Index() == i {
			m, cmd := field.Update(msg)
			g.selector.Set(i, m.(Field))
			cmds = append(cmds, cmd)
		}
		m, cmd := field.Update(updateFieldMsg{})
		g.selector.Set(i, m.(Field))
		cmds = append(cmds, cmd)
		return true
	})

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.WithHeight(max(g.height, min(g.fullHeight(), msg.Height-1)))
	case nextFieldMsg:
		cmds = append(cmds, g.nextField()...)
	case prevFieldMsg:
		cmds = append(cmds, g.prevField()...)
	}

	g.buildView()

	return g, tea.Batch(cmds...)
}

// height returns the full height of the group.
func (g *Group) fullHeight() int {
	height := g.selector.Total()
	g.selector.Range(func(_ int, field Field) bool {
		height += lipgloss.Height(field.View())
		return true
	})
	return height
}

func (g *Group) getContent() (int, string) {
	var fields strings.Builder
	offset := 0
	gap := "\n\n"

	// if the focused field is requesting it be zoomed, only show that field.
	if g.selector.Selected().Zoom() {
		g.selector.Selected().WithHeight(g.height - 1)
		fields.WriteString(g.selector.Selected().View())
	} else {
		g.selector.Range(func(i int, field Field) bool {
			fields.WriteString(field.View())
			if i == g.selector.Index() {
				offset = lipgloss.Height(fields.String()) - lipgloss.Height(field.View())
			}
			if i < g.selector.Total()-1 {
				fields.WriteString(gap)
			}
			return true
		})
	}

	return offset, fields.String() + "\n"
}

func (g *Group) buildView() {
	offset, content := g.getContent()

	g.viewport.SetContent(content)
	g.viewport.SetYOffset(offset)
}

// View renders the group.
func (g *Group) View() string {
	var view strings.Builder
	view.WriteString(g.viewport.View())
	view.WriteString(g.Footer())
	return view.String()
}

// Content renders the group's content only (no footer).
func (g *Group) Content() string {
	_, content := g.getContent()
	return content
}

// Footer renders the group's footer only (no content).
func (g *Group) Footer() string {
	var view strings.Builder
	view.WriteRune('\n')
	errors := g.Errors()
	if g.showHelp && len(errors) <= 0 {
		view.WriteString(g.help.ShortHelpView(g.selector.Selected().KeyBinds()))
	}
	if g.showErrors {
		for _, err := range errors {
			view.WriteString(ThemeCharm().Focused.ErrorMessage.Render(err.Error()))
		}
	}
	return view.String()
}
