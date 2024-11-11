package huh

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Note is a note field.
//
// A note is responsible for displaying information to the user. Use it to
// provide context around a different field. Generally, the notes are not
// interacted with unless the note has a next button `Next(true)`.
type Note struct {
	id int

	title       Eval[string]
	description Eval[string]
	nextLabel   string

	focused        bool
	showNextButton bool
	skip           bool

	accessible bool
	height     int
	width      int

	theme  *Theme
	keymap NoteKeyMap
}

// NewNote creates a new note field.
//
// A note is responsible for displaying information to the user. Use it to
// provide context around a different field. Generally, the notes are not
// interacted with unless the note has a next button `Next(true)`.
func NewNote() *Note {
	return &Note{
		id:             nextID(),
		showNextButton: false,
		skip:           true,
		nextLabel:      "Next",
		title:          Eval[string]{cache: make(map[uint64]string)},
		description:    Eval[string]{cache: make(map[uint64]string)},
	}
}

// Title sets the note field's title.
//
// This title will be static, for dynamic titles use `TitleFunc`.
func (n *Note) Title(title string) *Note {
	n.title.val = title
	n.title.fn = nil
	return n
}

// TitleFunc sets the title func of the note field.
//
// The TitleFunc will be re-evaluated when the binding of the TitleFunc changes.
// This is useful when you want to display dynamic content and update the title
// of a note when another part of your form changes.
//
// See README.md#Dynamic for more usage information.
func (n *Note) TitleFunc(f func() string, bindings any) *Note {
	n.title.fn = f
	n.title.bindings = bindings
	return n
}

// Description sets the note field's description.
//
// This description will be static, for dynamic descriptions use `DescriptionFunc`.
func (n *Note) Description(description string) *Note {
	n.description.val = description
	n.description.fn = nil
	return n
}

// DescriptionFunc sets the description func of the note field.
//
// The DescriptionFunc will be re-evaluated when the binding of the
// DescriptionFunc changes. This is useful when you want to display dynamic
// content and update the description of a note when another part of your form
// changes.
//
// For example, you can make a dynamic markdown preview with the following Form & Group.
//
//	huh.NewText().Title("Markdown").Value(&md),
//	huh.NewNote().Height(20).Title("Preview").
//	  DescriptionFunc(func() string {
//	      return md
//	  }, &md),
//
// Notice the `binding` of the Note is the same as the `Value` of the Text field.
// This binds the two values together, so that when the `Value` of the Text
// field changes so does the Note description.
func (n *Note) DescriptionFunc(f func() string, bindings any) *Note {
	n.description.fn = f
	n.description.bindings = bindings
	return n
}

// Height sets the note field's height.
func (n *Note) Height(height int) *Note {
	n.height = height
	return n
}

// Next sets whether or not to show the next button.
//
//	Title
//	Description
//
//	[ Next ]
func (n *Note) Next(show bool) *Note {
	n.showNextButton = show
	return n
}

// NextLabel sets the next button label.
func (n *Note) NextLabel(label string) *Note {
	n.nextLabel = label
	return n
}

// Focus focuses the note field.
func (n *Note) Focus() tea.Cmd {
	n.focused = true
	return nil
}

// Blur blurs the note field.
func (n *Note) Blur() tea.Cmd {
	n.focused = false
	return nil
}

// Error returns the error of the note field.
func (n *Note) Error() error { return nil }

// Skip returns whether the note should be skipped or should be blocking.
func (n *Note) Skip() bool { return n.skip }

// Zoom returns whether the note should be zoomed.
func (n *Note) Zoom() bool { return false }

// KeyBinds returns the help message for the note field.
func (n *Note) KeyBinds() []key.Binding {
	return []key.Binding{
		n.keymap.Prev,
		n.keymap.Submit,
		n.keymap.Next,
	}
}

// Init initializes the note field.
func (n *Note) Init() tea.Cmd { return nil }

// Update updates the note field.
func (n *Note) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateFieldMsg:
		var cmds []tea.Cmd
		if ok, hash := n.title.shouldUpdate(); ok {
			n.title.bindingsHash = hash
			if !n.title.loadFromCache() {
				n.title.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateTitleMsg{id: n.id, title: n.title.fn(), hash: hash}
				})
			}
		}
		if ok, hash := n.description.shouldUpdate(); ok {
			n.description.bindingsHash = hash
			if !n.description.loadFromCache() {
				n.description.loading = true
				cmds = append(cmds, func() tea.Msg {
					return updateDescriptionMsg{id: n.id, description: n.description.fn(), hash: hash}
				})
			}
		}
		return n, tea.Batch(cmds...)
	case updateTitleMsg:
		if msg.id == n.id && msg.hash == n.title.bindingsHash {
			n.title.update(msg.title)
		}
	case updateDescriptionMsg:
		if msg.id == n.id && msg.hash == n.description.bindingsHash {
			n.description.update(msg.description)
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, n.keymap.Prev):
			return n, PrevField
		case key.Matches(msg, n.keymap.Next, n.keymap.Submit):
			return n, NextField
		}
		return n, NextField
	}
	return n, nil
}

func (n *Note) activeStyles() *FieldStyles {
	theme := n.theme
	if theme == nil {
		theme = ThemeCharm()
	}
	if n.focused {
		return &theme.Focused
	}
	return &theme.Blurred
}

// View renders the note field.
func (n *Note) View() string {
	styles := n.activeStyles()
	sb := strings.Builder{}

	if n.title.val != "" || n.title.fn != nil {
		sb.WriteString(styles.NoteTitle.Render(n.title.val))
	}
	if n.description.val != "" || n.description.fn != nil {
		sb.WriteString("\n")
		sb.WriteString(render(n.description.val))
	}
	if n.showNextButton {
		sb.WriteString(styles.Next.Render(n.nextLabel))
	}
	return styles.Card.Height(n.height).Render(sb.String())
}

// Run runs the note field.
func (n *Note) Run() error {
	if n.accessible {
		return n.runAccessible()
	}
	return Run(n)
}

// runAccessible runs an accessible note field.
func (n *Note) runAccessible() error {
	if n.title.val != "" {
		fmt.Println(n.title.val)
		fmt.Println()
	}

	fmt.Println(n.description.val)
	fmt.Println()
	return nil
}

// WithTheme sets the theme on a note field.
func (n *Note) WithTheme(theme *Theme) Field {
	if n.theme != nil {
		return n
	}
	n.theme = theme
	return n
}

// WithKeyMap sets the keymap on a note field.
func (n *Note) WithKeyMap(k *KeyMap) Field {
	n.keymap = k.Note
	return n
}

// WithAccessible sets the accessible mode of the note field.
func (n *Note) WithAccessible(accessible bool) Field {
	n.accessible = accessible
	return n
}

// WithWidth sets the width of the note field.
func (n *Note) WithWidth(width int) Field {
	n.width = width
	return n
}

// WithHeight sets the height of the note field.
func (n *Note) WithHeight(height int) Field {
	n.Height(height)
	return n
}

// WithPosition sets the position information of the note field.
func (n *Note) WithPosition(p FieldPosition) Field {
	// if the note is the only field on the screen,
	// we shouldn't skip the entire group.
	if p.Field == p.FirstField && p.Field == p.LastField {
		n.skip = false
	}
	n.keymap.Prev.SetEnabled(!p.IsFirst())
	n.keymap.Next.SetEnabled(!p.IsLast())
	n.keymap.Submit.SetEnabled(p.IsLast())
	return n
}

// GetValue satisfies the Field interface, notes do not have values.
func (n *Note) GetValue() any { return nil }

// GetKey satisfies the Field interface, notes do not have keys.
func (n *Note) GetKey() string { return "" }

func render(input string) string {
	var result strings.Builder
	var italic, bold, codeblock bool
	var escape bool

	for _, char := range input {
		if escape || codeblock {
			result.WriteRune(char)
			escape = false
			continue
		}
		switch char {
		case '\\':
			escape = true
		case '_':
			if !italic {
				result.WriteString("\033[3m")
				italic = true
			} else {
				result.WriteString("\033[23m")
				italic = false
			}
		case '*':
			if !bold {
				result.WriteString("\033[1m")
				bold = true
			} else {
				result.WriteString("\033[22m")
				bold = false
			}
		case '`':
			if !codeblock {
				result.WriteString("\033[0;37;40m")
				result.WriteString(" ")
				codeblock = true
			} else {
				result.WriteString(" ")
				result.WriteString("\033[0m")
				codeblock = false

				if bold {
					result.WriteString("\033[1m")
				}
				if italic {
					result.WriteString("\033[3m")
				}
			}
		default:
			result.WriteRune(char)
		}
	}

	// Reset any open formatting
	result.WriteString("\033[0m")

	return result.String()
}
