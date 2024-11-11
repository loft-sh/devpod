package huh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh/internal/selector"
)

const defaultWidth = 80

// Internal ID management. Used during animating to ensure that frame messages
// are received only by spinner components that sent them.
var (
	lastID int
	idMtx  sync.Mutex
)

// Return the next ID we should use on the Model.
func nextID() int {
	idMtx.Lock()
	defer idMtx.Unlock()
	lastID++
	return lastID
}

// FormState represents the current state of the form.
type FormState int

const (
	// StateNormal is when the user is completing the form.
	StateNormal FormState = iota

	// StateCompleted is when the user has completed the form.
	StateCompleted

	// StateAborted is when the user has aborted the form.
	StateAborted
)

// ErrUserAborted is the error returned when a user exits the form before submitting.
var ErrUserAborted = errors.New("user aborted")

// ErrTimeout is the error returned when the timeout is reached.
var ErrTimeout = errors.New("timeout")

// ErrTimeoutUnsupported is the error returned when timeout is used while in accessible mode.
var ErrTimeoutUnsupported = errors.New("timeout is not supported in accessible mode")

// Form is a collection of groups that are displayed one at a time on a "page".
//
// The form can navigate between groups and is complete once all the groups are
// complete.
type Form struct {
	// collection of groups
	selector *selector.Selector[*Group]

	results map[string]any

	// callbacks
	SubmitCmd tea.Cmd
	CancelCmd tea.Cmd

	State FormState

	// whether or not to use bubble tea rendering for accessibility
	// purposes, if true, the form will render with basic prompting primitives
	// to be more accessible to screen readers.
	accessible bool

	quitting bool
	aborted  bool

	// options
	width      int
	height     int
	keymap     *KeyMap
	timeout    time.Duration
	teaOptions []tea.ProgramOption

	layout Layout
}

// NewForm returns a form with the given groups and default themes and
// keybindings.
//
// Use With* methods to customize the form with options, such as setting
// different themes and keybindings.
func NewForm(groups ...*Group) *Form {
	selector := selector.NewSelector(groups)

	f := &Form{
		selector: selector,
		keymap:   NewDefaultKeyMap(),
		results:  make(map[string]any),
		layout:   LayoutDefault,
		teaOptions: []tea.ProgramOption{
			tea.WithOutput(os.Stderr),
		},
	}

	// NB: If dynamic forms come into play this will need to be applied when
	// groups and fields are added.
	f.WithKeyMap(f.keymap)
	f.WithWidth(f.width)
	f.WithHeight(f.height)
	f.UpdateFieldPositions()

	if os.Getenv("TERM") == "dumb" {
		f.WithWidth(defaultWidth)
		f.WithAccessible(true)
	}

	return f
}

// Field is a primitive of a form.
//
// A field represents a single input control on a form such as a text input,
// confirm button, select option, etc...
//
// Each field implements the Bubble Tea Model interface.
type Field interface {
	// Bubble Tea Model
	Init() tea.Cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() string

	// Bubble Tea Events
	Blur() tea.Cmd
	Focus() tea.Cmd

	// Errors and Validation
	Error() error

	// Run runs the field individually.
	Run() error

	// Skip returns whether this input should be skipped or not.
	Skip() bool

	// Zoom returns whether this input should be zoomed or not.
	// Zoom allows the field to take focus of the group / form height.
	Zoom() bool

	// KeyBinds returns help keybindings.
	KeyBinds() []key.Binding

	// WithTheme sets the theme on a field.
	WithTheme(*Theme) Field

	// WithAccessible sets whether the field should run in accessible mode.
	WithAccessible(bool) Field

	// WithKeyMap sets the keymap on a field.
	WithKeyMap(*KeyMap) Field

	// WithWidth sets the width of a field.
	WithWidth(int) Field

	// WithHeight sets the height of a field.
	WithHeight(int) Field

	// WithPosition tells the field the index of the group and position it is in.
	WithPosition(FieldPosition) Field

	// GetKey returns the field's key.
	GetKey() string

	// GetValue returns the field's value.
	GetValue() any
}

// FieldPosition is positional information about the given field and form.
type FieldPosition struct {
	Group      int
	Field      int
	FirstField int
	LastField  int
	GroupCount int
	FirstGroup int
	LastGroup  int
}

// IsFirst returns whether a field is the form's first field.
func (p FieldPosition) IsFirst() bool {
	return p.Field == p.FirstField && p.Group == p.FirstGroup
}

// IsLast returns whether a field is the form's last field.
func (p FieldPosition) IsLast() bool {
	return p.Field == p.LastField && p.Group == p.LastGroup
}

// nextGroupMsg is a message to move to the next group.
type nextGroupMsg struct{}

// prevGroupMsg is a message to move to the previous group.
type prevGroupMsg struct{}

// nextGroup is the command to move to the next group.
func nextGroup() tea.Msg {
	return nextGroupMsg{}
}

// prevGroup is the command to move to the previous group.
func prevGroup() tea.Msg {
	return prevGroupMsg{}
}

// WithAccessible sets the form to run in accessible mode to avoid redrawing the
// views which makes it easier for screen readers to read and describe the form.
//
// This avoids using the Bubble Tea renderer and instead simply uses basic
// terminal prompting to gather input which degrades the user experience but
// provides accessibility.
func (f *Form) WithAccessible(accessible bool) *Form {
	f.accessible = accessible
	return f
}

// WithShowHelp sets whether or not the form should show help.
//
// This allows the form groups and field to show what keybindings are available
// to the user.
func (f *Form) WithShowHelp(v bool) *Form {
	f.selector.Range(func(_ int, group *Group) bool {
		group.WithShowHelp(v)
		return true
	})
	return f
}

// WithShowErrors sets whether or not the form should show errors.
//
// This allows the form groups and fields to show errors when the Validate
// function returns an error.
func (f *Form) WithShowErrors(v bool) *Form {
	f.selector.Range(func(_ int, group *Group) bool {
		group.WithShowErrors(v)
		return true
	})
	return f
}

// WithTheme sets the theme on a form.
//
// This allows all groups and fields to be themed consistently, however themes
// can be applied to each group and field individually for more granular
// control.
func (f *Form) WithTheme(theme *Theme) *Form {
	if theme == nil {
		return f
	}
	f.selector.Range(func(_ int, group *Group) bool {
		group.WithTheme(theme)
		return true
	})
	return f
}

// WithKeyMap sets the keymap on a form.
//
// This allows customization of the form key bindings.
func (f *Form) WithKeyMap(keymap *KeyMap) *Form {
	if keymap == nil {
		return f
	}
	f.keymap = keymap
	f.selector.Range(func(_ int, group *Group) bool {
		group.WithKeyMap(keymap)
		return true
	})
	f.UpdateFieldPositions()
	return f
}

// WithWidth sets the width of a form.
//
// This allows all groups and fields to be sized consistently, however width
// can be applied to each group and field individually for more granular
// control.
func (f *Form) WithWidth(width int) *Form {
	if width <= 0 {
		return f
	}
	f.width = width
	f.selector.Range(func(_ int, group *Group) bool {
		width := f.layout.GroupWidth(f, group, width)
		group.WithWidth(width)
		return true
	})
	return f
}

// WithHeight sets the height of a form.
func (f *Form) WithHeight(height int) *Form {
	if height <= 0 {
		return f
	}
	f.height = height
	f.selector.Range(func(_ int, group *Group) bool {
		group.WithHeight(height)
		return true
	})
	return f
}

// WithOutput sets the io.Writer to output the form.
func (f *Form) WithOutput(w io.Writer) *Form {
	f.teaOptions = append(f.teaOptions, tea.WithOutput(w))
	return f
}

// WithInput sets the io.Reader to the input form.
func (f *Form) WithInput(r io.Reader) *Form {
	f.teaOptions = append(f.teaOptions, tea.WithInput(r))
	return f
}

// WithTimeout sets the duration for the form to be killed.
func (f *Form) WithTimeout(t time.Duration) *Form {
	f.timeout = t
	return f
}

// WithProgramOptions sets the tea options of the form.
func (f *Form) WithProgramOptions(opts ...tea.ProgramOption) *Form {
	f.teaOptions = opts
	return f
}

// WithLayout sets the layout on a form.
//
// This allows customization of the form group layout.
func (f *Form) WithLayout(layout Layout) *Form {
	f.layout = layout
	return f
}

// UpdateFieldPositions sets the position on all the fields.
func (f *Form) UpdateFieldPositions() *Form {
	firstGroup := 0
	lastGroup := f.selector.Total() - 1

	// determine the first non-hidden group.
	f.selector.Range(func(_ int, g *Group) bool {
		if !f.isGroupHidden(g) {
			return false
		}
		firstGroup++
		return true
	})

	// determine the last non-hidden group.
	f.selector.ReverseRange(func(_ int, g *Group) bool {
		if !f.isGroupHidden(g) {
			return false
		}
		lastGroup--
		return true
	})

	f.selector.Range(func(g int, group *Group) bool {
		// determine the first non-skippable field.
		var firstField int
		group.selector.Range(func(_ int, field Field) bool {
			if !field.Skip() || group.selector.Total() == 1 {
				return false
			}
			firstField++
			return true
		})

		// determine the last non-skippable field.
		var lastField int
		group.selector.ReverseRange(func(i int, field Field) bool {
			lastField = i
			if !field.Skip() || group.selector.Total() == 1 {
				return false
			}
			return true
		})

		group.selector.Range(func(i int, field Field) bool {
			field.WithPosition(FieldPosition{
				Group:      g,
				Field:      i,
				FirstField: firstField,
				LastField:  lastField,
				FirstGroup: firstGroup,
				LastGroup:  lastGroup,
			})
			return true
		})

		return true
	})
	return f
}

// Errors returns the current groups' errors.
func (f *Form) Errors() []error {
	return f.selector.Selected().Errors()
}

// Help returns the current groups' help.
func (f *Form) Help() help.Model {
	return f.selector.Selected().help
}

// KeyBinds returns the current fields' keybinds.
func (f *Form) KeyBinds() []key.Binding {
	group := f.selector.Selected()
	return group.selector.Selected().KeyBinds()
}

// Get returns a result from the form.
func (f *Form) Get(key string) any {
	return f.results[key]
}

// GetString returns a result as a string from the form.
func (f *Form) GetString(key string) string {
	v, ok := f.results[key].(string)
	if !ok {
		return ""
	}
	return v
}

// GetInt returns a result as a int from the form.
func (f *Form) GetInt(key string) int {
	v, ok := f.results[key].(int)
	if !ok {
		return 0
	}
	return v
}

// GetBool returns a result as a string from the form.
func (f *Form) GetBool(key string) bool {
	v, ok := f.results[key].(bool)
	if !ok {
		return false
	}
	return v
}

// NextGroup moves the form to the next group.
func (f *Form) NextGroup() tea.Cmd {
	_, cmd := f.Update(nextGroup())
	return cmd
}

// PrevGroup moves the form to the next group.
func (f *Form) PrevGroup() tea.Cmd {
	_, cmd := f.Update(prevGroup())
	return cmd
}

// NextField moves the form to the next field.
func (f *Form) NextField() tea.Cmd {
	_, cmd := f.Update(NextField())
	return cmd
}

// NextField moves the form to the next field.
func (f *Form) PrevField() tea.Cmd {
	_, cmd := f.Update(PrevField())
	return cmd
}

// Init initializes the form.
func (f *Form) Init() tea.Cmd {
	cmds := make([]tea.Cmd, f.selector.Total())
	f.selector.Range(func(i int, group *Group) bool {
		if i == 0 {
			group.active = true
		}
		cmds[i] = group.Init()
		return true
	})

	if f.isGroupHidden(f.selector.Selected()) {
		cmds = append(cmds, nextGroup)
	}

	return tea.Batch(cmds...)
}

// Update updates the form.
func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If the form is aborted or completed there's no need to update it.
	if f.State != StateNormal {
		return f, nil
	}

	group := f.selector.Selected()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if f.width > 0 {
			break
		}
		f.selector.Range(func(_ int, group *Group) bool {
			width := f.layout.GroupWidth(f, group, msg.Width)
			group.WithWidth(width)
			return true
		})
		if f.height > 0 {
			break
		}
		f.selector.Range(func(_ int, group *Group) bool {
			if group.fullHeight() > msg.Height {
				group.WithHeight(msg.Height)
			}
			return true
		})
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, f.keymap.Quit):
			f.aborted = true
			f.quitting = true
			f.State = StateAborted
			return f, f.CancelCmd
		}

	case nextFieldMsg:
		// Form is progressing to the next field, let's save the value of the current field.
		field := group.selector.Selected()
		f.results[field.GetKey()] = field.GetValue()

	case nextGroupMsg:
		if len(group.Errors()) > 0 {
			return f, nil
		}

		submit := func() (tea.Model, tea.Cmd) {
			f.quitting = true
			f.State = StateCompleted
			return f, f.SubmitCmd
		}

		if f.selector.OnLast() {
			return submit()
		}

		for i := f.selector.Index() + 1; i < f.selector.Total(); i++ {
			if !f.isGroupHidden(f.selector.Get(i)) {
				f.selector.SetIndex(i)
				break
			}
			// all subsequent groups are hidden, so we must act as
			// if we were in the last one.
			if i == f.selector.Total()-1 {
				return submit()
			}
		}
		f.selector.Selected().active = true
		return f, f.selector.Selected().Init()

	case prevGroupMsg:
		if len(group.Errors()) > 0 {
			return f, nil
		}

		for i := f.selector.Index() - 1; i >= 0; i-- {
			if !f.isGroupHidden(f.selector.Get(i)) {
				f.selector.SetIndex(i)
				break
			}
		}

		f.selector.Selected().active = true
		return f, f.selector.Selected().Init()
	}

	m, cmd := group.Update(msg)
	f.selector.Set(f.selector.Index(), m.(*Group))

	// A user input a key, this could hide or show other groups,
	// let's update all of their positions.
	switch msg.(type) {
	case tea.KeyMsg:
		f.UpdateFieldPositions()
	}

	return f, cmd
}

func (f *Form) isGroupHidden(group *Group) bool {
	hide := group.hide
	if hide == nil {
		return false
	}
	return hide()
}

// View renders the form.
func (f *Form) View() string {
	if f.quitting {
		return ""
	}

	return f.layout.View(f)
}

// Run runs the form.
func (f *Form) Run() error {
	return f.RunWithContext(context.Background())
}

// RunWithContext runs the form with the given context.
func (f *Form) RunWithContext(ctx context.Context) error {
	f.SubmitCmd = tea.Quit
	f.CancelCmd = tea.Quit

	if f.selector.Total() == 0 {
		return nil
	}

	if f.accessible {
		return f.runAccessible()
	}

	return f.run(ctx)
}

// run runs the form in normal mode.
func (f *Form) run(ctx context.Context) error {
	if f.timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, f.timeout)
		defer cancel()
		f.teaOptions = append(f.teaOptions, tea.WithContext(ctx), tea.WithReportFocus())
	} else {
		f.teaOptions = append(f.teaOptions, tea.WithContext(ctx), tea.WithReportFocus())
	}

	m, err := tea.NewProgram(f, f.teaOptions...).Run()
	if m.(*Form).aborted {
		return ErrUserAborted
	}
	if errors.Is(err, tea.ErrProgramKilled) {
		return ErrTimeout
	}
	if err != nil {
		return fmt.Errorf("huh: %w", err)
	}
	return nil
}

// runAccessible runs the form in accessible mode.
func (f *Form) runAccessible() error {
	// Timeouts are not supported in this mode.
	if f.timeout > 0 {
		return ErrTimeoutUnsupported
	}

	f.selector.Range(func(_ int, group *Group) bool {
		group.selector.Range(func(_ int, field Field) bool {
			field.Init()
			field.Focus()
			_ = field.WithAccessible(true).Run()
			return true
		})
		return true
	})

	return nil
}
