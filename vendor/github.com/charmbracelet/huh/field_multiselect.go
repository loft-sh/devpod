package huh

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh/accessibility"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelect is a form multi-select field.
type MultiSelect[T comparable] struct {
	accessor Accessor[[]T]
	key      string
	id       int

	// customization
	title           Eval[string]
	description     Eval[string]
	options         Eval[[]Option[T]]
	filterable      bool
	filteredOptions []Option[T]
	limit           int
	height          int

	// error handling
	validate func([]T) error
	err      error

	// state
	cursor    int
	focused   bool
	filtering bool
	filter    textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model

	// options
	width      int
	accessible bool
	theme      *Theme
	keymap     MultiSelectKeyMap
}

// NewMultiSelect returns a new multi-select field.
func NewMultiSelect[T comparable]() *MultiSelect[T] {
	filter := textinput.New()
	filter.Prompt = "/"

	s := spinner.New(spinner.WithSpinner(spinner.Line))

	return &MultiSelect[T]{
		accessor:    &EmbeddedAccessor[[]T]{},
		validate:    func([]T) error { return nil },
		filtering:   false,
		filter:      filter,
		id:          nextID(),
		options:     Eval[[]Option[T]]{cache: make(map[uint64][]Option[T])},
		title:       Eval[string]{cache: make(map[uint64]string)},
		description: Eval[string]{cache: make(map[uint64]string)},
		spinner:     s,
		filterable:  true,
	}
}

// Value sets the value of the multi-select field.
func (m *MultiSelect[T]) Value(value *[]T) *MultiSelect[T] {
	return m.Accessor(NewPointerAccessor(value))
}

// Accessor sets the accessor of the input field.
func (m *MultiSelect[T]) Accessor(accessor Accessor[[]T]) *MultiSelect[T] {
	m.accessor = accessor
	for i, o := range m.options.val {
		for _, v := range m.accessor.Get() {
			if o.Value == v {
				m.options.val[i].selected = true
				break
			}
		}
	}
	return m
}

// Key sets the key of the select field which can be used to retrieve the value
// after submission.
func (m *MultiSelect[T]) Key(key string) *MultiSelect[T] {
	m.key = key
	return m
}

// Title sets the title of the multi-select field.
func (m *MultiSelect[T]) Title(title string) *MultiSelect[T] {
	m.title.val = title
	m.title.fn = nil
	return m
}

// TitleFunc sets the title func of the multi-select field.
func (m *MultiSelect[T]) TitleFunc(f func() string, bindings any) *MultiSelect[T] {
	m.title.fn = f
	m.title.bindings = bindings
	return m
}

// Description sets the description of the multi-select field.
func (m *MultiSelect[T]) Description(description string) *MultiSelect[T] {
	m.description.val = description
	return m
}

// DescriptionFunc sets the description func of the multi-select field.
func (m *MultiSelect[T]) DescriptionFunc(f func() string, bindings any) *MultiSelect[T] {
	m.description.fn = f
	m.description.bindings = bindings
	return m
}

// Options sets the options of the multi-select field.
func (m *MultiSelect[T]) Options(options ...Option[T]) *MultiSelect[T] {
	if len(options) <= 0 {
		return m
	}

	for i, o := range options {
		for _, v := range m.accessor.Get() {
			if o.Value == v {
				options[i].selected = true
				break
			}
		}
	}
	m.options.val = options
	m.filteredOptions = options
	m.updateViewportHeight()
	return m
}

// OptionsFunc sets the options func of the multi-select field.
func (m *MultiSelect[T]) OptionsFunc(f func() []Option[T], bindings any) *MultiSelect[T] {
	m.options.fn = f
	m.options.bindings = bindings
	m.filteredOptions = make([]Option[T], 0)
	// If there is no height set, we should attach a static height since these
	// options are possibly dynamic.
	if m.height <= 0 {
		m.height = defaultHeight
		m.updateViewportHeight()
	}
	return m
}

// Filterable sets the multi-select field as filterable.
func (m *MultiSelect[T]) Filterable(filterable bool) *MultiSelect[T] {
	m.filterable = filterable
	return m
}

// Filtering sets the filtering state of the multi-select field.
func (m *MultiSelect[T]) Filtering(filtering bool) *MultiSelect[T] {
	m.filtering = filtering
	m.filter.Focus()
	return m
}

// Limit sets the limit of the multi-select field.
func (m *MultiSelect[T]) Limit(limit int) *MultiSelect[T] {
	m.limit = limit
	m.setSelectAllHelp()
	return m
}

// Height sets the height of the multi-select field.
func (m *MultiSelect[T]) Height(height int) *MultiSelect[T] {
	// What we really want to do is set the height of the viewport, but we
	// need a theme applied before we can calcualate its height.
	m.height = height
	m.updateViewportHeight()
	return m
}

// Validate sets the validation function of the multi-select field.
func (m *MultiSelect[T]) Validate(validate func([]T) error) *MultiSelect[T] {
	m.validate = validate
	return m
}

// Error returns the error of the multi-select field.
func (m *MultiSelect[T]) Error() error {
	return m.err
}

// Skip returns whether the multiselect should be skipped or should be blocking.
func (*MultiSelect[T]) Skip() bool {
	return false
}

// Zoom returns whether the multiselect should be zoomed.
func (*MultiSelect[T]) Zoom() bool {
	return false
}

// Focus focuses the multi-select field.
func (m *MultiSelect[T]) Focus() tea.Cmd {
	m.updateValue()
	m.focused = true
	return nil
}

// Blur blurs the multi-select field.
func (m *MultiSelect[T]) Blur() tea.Cmd {
	m.updateValue()
	m.focused = false
	return nil
}

// KeyBinds returns the help message for the multi-select field.
func (m *MultiSelect[T]) KeyBinds() []key.Binding {
	binds := []key.Binding{
		m.keymap.Toggle,
		m.keymap.Up,
		m.keymap.Down,
	}
	if m.filterable {
		binds = append(
			binds,
			m.keymap.Filter,
			m.keymap.SetFilter,
			m.keymap.ClearFilter,
		)
	}
	binds = append(
		binds,
		m.keymap.Prev,
		m.keymap.Submit,
		m.keymap.Next,
		m.keymap.SelectAll,
		m.keymap.SelectNone,
	)
	return binds
}

// Init initializes the multi-select field.
func (m *MultiSelect[T]) Init() tea.Cmd {
	return nil
}

// Update updates the multi-select field.
func (m *MultiSelect[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Enforce height on the viewport during update as we need themes to
	// be applied before we can calculate the height.
	m.updateViewportHeight()

	var cmd tea.Cmd
	if m.filtering {
		m.filter, cmd = m.filter.Update(msg)
		m.setSelectAllHelp()
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case updateFieldMsg:
		var fieldCmds []tea.Cmd
		if ok, hash := m.title.shouldUpdate(); ok {
			m.title.bindingsHash = hash
			if !m.title.loadFromCache() {
				m.title.loading = true
				fieldCmds = append(fieldCmds, func() tea.Msg {
					return updateTitleMsg{id: m.id, title: m.title.fn(), hash: hash}
				})
			}
		}
		if ok, hash := m.description.shouldUpdate(); ok {
			m.description.bindingsHash = hash
			if !m.description.loadFromCache() {
				m.description.loading = true
				fieldCmds = append(fieldCmds, func() tea.Msg {
					return updateDescriptionMsg{id: m.id, description: m.description.fn(), hash: hash}
				})
			}
		}
		if ok, hash := m.options.shouldUpdate(); ok {
			m.options.bindingsHash = hash
			if m.options.loadFromCache() {
				m.filteredOptions = m.options.val
				m.updateValue()
				m.cursor = clamp(m.cursor, 0, len(m.filteredOptions)-1)
			} else {
				m.options.loading = true
				m.options.loadingStart = time.Now()
				fieldCmds = append(fieldCmds, func() tea.Msg {
					return updateOptionsMsg[T]{id: m.id, options: m.options.fn(), hash: hash}
				}, m.spinner.Tick)
			}
		}

		return m, tea.Batch(fieldCmds...)

	case spinner.TickMsg:
		if !m.options.loading {
			break
		}
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case updateTitleMsg:
		if msg.id == m.id && msg.hash == m.title.bindingsHash {
			m.title.update(msg.title)
		}
	case updateDescriptionMsg:
		if msg.id == m.id && msg.hash == m.description.bindingsHash {
			m.description.update(msg.description)
		}
	case updateOptionsMsg[T]:
		if msg.id == m.id && msg.hash == m.options.bindingsHash {
			m.options.update(msg.options)
			// since we're updating the options, we need to reset the cursor.
			m.filteredOptions = m.options.val
			m.updateValue()
			m.cursor = clamp(m.cursor, 0, len(m.filteredOptions)-1)
		}
	case tea.KeyMsg:
		m.err = nil
		switch {
		case key.Matches(msg, m.keymap.Filter):
			m.setFilter(true)
			return m, m.filter.Focus()
		case key.Matches(msg, m.keymap.SetFilter):
			if len(m.filteredOptions) <= 0 {
				m.filter.SetValue("")
				m.filteredOptions = m.options.val
			}
			m.setFilter(false)
		case key.Matches(msg, m.keymap.ClearFilter):
			m.filter.SetValue("")
			m.filteredOptions = m.options.val
			m.setFilter(false)
		case key.Matches(msg, m.keymap.Up):
			// FIXME: should use keys in keymap
			if m.filtering && msg.String() == "k" {
				break
			}

			m.cursor = max(m.cursor-1, 0)
			if m.cursor < m.viewport.YOffset {
				m.viewport.SetYOffset(m.cursor)
			}
		case key.Matches(msg, m.keymap.Down):
			// FIXME: should use keys in keymap
			if m.filtering && msg.String() == "j" {
				break
			}

			m.cursor = min(m.cursor+1, len(m.filteredOptions)-1)
			if m.cursor >= m.viewport.YOffset+m.viewport.Height {
				m.viewport.LineDown(1)
			}
		case key.Matches(msg, m.keymap.GotoTop):
			if m.filtering {
				break
			}
			m.cursor = 0
			m.viewport.GotoTop()
		case key.Matches(msg, m.keymap.GotoBottom):
			if m.filtering {
				break
			}
			m.cursor = len(m.filteredOptions) - 1
			m.viewport.GotoBottom()
		case key.Matches(msg, m.keymap.HalfPageUp):
			m.cursor = max(m.cursor-m.viewport.Height/2, 0)
			m.viewport.HalfViewUp()
		case key.Matches(msg, m.keymap.HalfPageDown):
			m.cursor = min(m.cursor+m.viewport.Height/2, len(m.filteredOptions)-1)
			m.viewport.HalfViewDown()
		case key.Matches(msg, m.keymap.Toggle) && !m.filtering:
			for i, option := range m.options.val {
				if option.Key == m.filteredOptions[m.cursor].Key {
					if !m.options.val[m.cursor].selected && m.limit > 0 && m.numSelected() >= m.limit {
						break
					}
					selected := m.options.val[i].selected
					m.options.val[i].selected = !selected
					m.filteredOptions[m.cursor].selected = !selected
				}
			}
			m.setSelectAllHelp()
			m.updateValue()
		case key.Matches(msg, m.keymap.SelectAll, m.keymap.SelectNone) && m.limit <= 0:
			selected := false

			for _, option := range m.filteredOptions {
				if !option.selected {
					selected = true
					break
				}
			}

			for i, option := range m.options.val {
				for j := range m.filteredOptions {
					if option.Key == m.filteredOptions[j].Key {
						m.options.val[i].selected = selected
						m.filteredOptions[j].selected = selected
						break
					}
				}
			}
			m.setSelectAllHelp()
			m.updateValue()
		case key.Matches(msg, m.keymap.Prev):
			m.updateValue()
			m.err = m.validate(m.accessor.Get())
			if m.err != nil {
				return m, nil
			}
			return m, PrevField
		case key.Matches(msg, m.keymap.Next, m.keymap.Submit):
			m.updateValue()
			m.err = m.validate(m.accessor.Get())
			if m.err != nil {
				return m, nil
			}
			return m, NextField
		}

		if m.filtering {
			m.filteredOptions = m.options.val
			if m.filter.Value() != "" {
				m.filteredOptions = nil
				for _, option := range m.options.val {
					if m.filterFunc(option.Key) {
						m.filteredOptions = append(m.filteredOptions, option)
					}
				}
			}
			if len(m.filteredOptions) > 0 {
				m.cursor = min(m.cursor, len(m.filteredOptions)-1)
				m.viewport.SetYOffset(clamp(m.cursor, 0, len(m.filteredOptions)-m.viewport.Height))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// updateViewportHeight updates the viewport size according to the Height setting
// on this multi-select field.
func (m *MultiSelect[T]) updateViewportHeight() {
	// If no height is set size the viewport to the number of options.
	if m.height <= 0 {
		m.viewport.Height = len(m.options.val)
		return
	}

	const minHeight = 1
	m.viewport.Height = max(minHeight, m.height-
		lipgloss.Height(m.titleView())-
		lipgloss.Height(m.descriptionView()))
}

// numSelected returns the total number of selected options.
func (m *MultiSelect[T]) numSelected() int {
	var count int
	for _, o := range m.options.val {
		if o.selected {
			count++
		}
	}
	return count
}

// numFilteredOptionsSelected returns the number of selected options with the
// current filter applied.
func (m *MultiSelect[T]) numFilteredSelected() int {
	var count int
	for _, o := range m.filteredOptions {
		if o.selected {
			count++
		}
	}
	return count
}

func (m *MultiSelect[T]) updateValue() {
	value := make([]T, 0)
	for _, option := range m.options.val {
		if option.selected {
			value = append(value, option.Value)
		}
	}
	m.accessor.Set(value)
	m.err = m.validate(m.accessor.Get())
}

func (m *MultiSelect[T]) activeStyles() *FieldStyles {
	theme := m.theme
	if theme == nil {
		theme = ThemeCharm()
	}
	if m.focused {
		return &theme.Focused
	}
	return &theme.Blurred
}

func (m *MultiSelect[T]) titleView() string {
	if m.title.val == "" {
		return ""
	}
	var (
		styles = m.activeStyles()
		sb     = strings.Builder{}
	)
	if m.filtering {
		sb.WriteString(m.filter.View())
	} else if m.filter.Value() != "" {
		sb.WriteString(styles.Title.Render(m.title.val) + styles.Description.Render("/"+m.filter.Value()))
	} else {
		sb.WriteString(styles.Title.Render(m.title.val))
	}
	if m.err != nil {
		sb.WriteString(styles.ErrorIndicator.String())
	}
	return sb.String()
}

func (m *MultiSelect[T]) descriptionView() string {
	return m.activeStyles().Description.Render(m.description.val)
}

func (m *MultiSelect[T]) optionsView() string {
	var (
		styles = m.activeStyles()
		c      = styles.MultiSelectSelector.String()
		sb     strings.Builder
	)

	if m.options.loading && time.Since(m.options.loadingStart) > spinnerShowThreshold {
		m.spinner.Style = m.activeStyles().MultiSelectSelector.UnsetString()
		sb.WriteString(m.spinner.View() + " Loading...")
		return sb.String()
	}

	for i, option := range m.filteredOptions {
		if m.cursor == i {
			sb.WriteString(c)
		} else {
			sb.WriteString(strings.Repeat(" ", lipgloss.Width(c)))
		}

		if m.filteredOptions[i].selected {
			sb.WriteString(styles.SelectedPrefix.String())
			sb.WriteString(styles.SelectedOption.Render(option.Key))
		} else {
			sb.WriteString(styles.UnselectedPrefix.String())
			sb.WriteString(styles.UnselectedOption.Render(option.Key))
		}
		if i < len(m.options.val)-1 {
			sb.WriteString("\n")
		}
	}

	for i := len(m.filteredOptions); i < len(m.options.val)-1; i++ {
		sb.WriteString("\n")
	}

	return sb.String()
}

// View renders the multi-select field.
func (m *MultiSelect[T]) View() string {
	styles := m.activeStyles()

	m.viewport.SetContent(m.optionsView())

	var sb strings.Builder
	if m.title.val != "" || m.title.fn != nil {
		sb.WriteString(m.titleView())
		sb.WriteString("\n")
	}
	if m.description.val != "" || m.description.fn != nil {
		sb.WriteString(m.descriptionView() + "\n")
	}
	sb.WriteString(m.viewport.View())
	return styles.Base.Render(sb.String())
}

func (m *MultiSelect[T]) printOptions() {
	styles := m.activeStyles()
	var sb strings.Builder
	sb.WriteString(styles.Title.Render(m.title.val))
	sb.WriteString("\n")

	for i, option := range m.options.val {
		if option.selected {
			sb.WriteString(styles.SelectedOption.Render(fmt.Sprintf("%d. %s %s", i+1, "✓", option.Key)))
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s %s", i+1, " ", option.Key))
		}
		sb.WriteString("\n")
	}

	fmt.Println(sb.String())
}

// setFilter sets the filter of the select field.
func (m *MultiSelect[T]) setFilter(filter bool) {
	m.filtering = filter
	m.keymap.SetFilter.SetEnabled(filter)
	m.keymap.Filter.SetEnabled(!filter)
	m.keymap.Next.SetEnabled(!filter)
	m.keymap.Submit.SetEnabled(!filter)
	m.keymap.Prev.SetEnabled(!filter)
	m.keymap.ClearFilter.SetEnabled(!filter && m.filter.Value() != "")
}

// filterFunc returns true if the option matches the filter.
func (m *MultiSelect[T]) filterFunc(option string) bool {
	// XXX: remove diacritics or allow customization of filter function.
	return strings.Contains(strings.ToLower(option), strings.ToLower(m.filter.Value()))
}

// setSelectAllHelp enables the appropriate select all or select none keybinding.
func (m *MultiSelect[T]) setSelectAllHelp() {
	if m.limit <= 0 {
		noneSelected := m.numFilteredSelected() <= 0
		allSelected := m.numFilteredSelected() > 0 && m.numFilteredSelected() < len(m.filteredOptions)
		selectAll := noneSelected || allSelected
		m.keymap.SelectAll.SetEnabled(selectAll)
		m.keymap.SelectNone.SetEnabled(!selectAll)
	}
}

// Run runs the multi-select field.
func (m *MultiSelect[T]) Run() error {
	if m.accessible {
		return m.runAccessible()
	}
	return Run(m)
}

// runAccessible() runs the multi-select field in accessible mode.
func (m *MultiSelect[T]) runAccessible() error {
	m.printOptions()
	styles := m.activeStyles()

	var choice int
	for {
		fmt.Printf("Select up to %d options. 0 to continue.\n", m.limit)

		choice = accessibility.PromptInt("Select: ", 0, len(m.options.val))
		if choice == 0 {
			m.updateValue()
			err := m.validate(m.accessor.Get())
			if err != nil {
				fmt.Println(err)
				continue
			}
			break
		}

		if !m.options.val[choice-1].selected && m.limit > 0 && m.numSelected() >= m.limit {
			fmt.Printf("You can't select more than %d options.\n", m.limit)
			continue
		}
		m.options.val[choice-1].selected = !m.options.val[choice-1].selected
		if m.options.val[choice-1].selected {
			fmt.Printf("Selected: %s\n\n", m.options.val[choice-1].Key)
		} else {
			fmt.Printf("Deselected: %s\n\n", m.options.val[choice-1].Key)
		}

		m.printOptions()
	}

	var values []string

	value := m.accessor.Get()
	for _, option := range m.options.val {
		if option.selected {
			value = append(value, option.Value)
			values = append(values, option.Key)
		}
	}
	m.accessor.Set(value)

	fmt.Println(styles.SelectedOption.Render("Selected:", strings.Join(values, ", ")+"\n"))
	return nil
}

// WithTheme sets the theme of the multi-select field.
func (m *MultiSelect[T]) WithTheme(theme *Theme) Field {
	if m.theme != nil {
		return m
	}
	m.theme = theme
	m.filter.Cursor.Style = theme.Focused.TextInput.Cursor
	m.filter.Cursor.TextStyle = theme.Focused.TextInput.CursorText
	m.filter.PromptStyle = theme.Focused.TextInput.Prompt
	m.filter.TextStyle = theme.Focused.TextInput.Text
	m.filter.PlaceholderStyle = theme.Focused.TextInput.Placeholder
	m.updateViewportHeight()
	return m
}

// WithKeyMap sets the keymap of the multi-select field.
func (m *MultiSelect[T]) WithKeyMap(k *KeyMap) Field {
	m.keymap = k.MultiSelect
	if !m.filterable {
		m.keymap.Filter.SetEnabled(false)
		m.keymap.ClearFilter.SetEnabled(false)
		m.keymap.SetFilter.SetEnabled(false)
	}
	return m
}

// WithAccessible sets the accessible mode of the multi-select field.
func (m *MultiSelect[T]) WithAccessible(accessible bool) Field {
	m.accessible = accessible
	return m
}

// WithWidth sets the width of the multi-select field.
func (m *MultiSelect[T]) WithWidth(width int) Field {
	m.width = width
	return m
}

// WithHeight sets the total height of the multi-select field. Including padding
// and help menu heights.
func (m *MultiSelect[T]) WithHeight(height int) Field {
	m.Height(height)
	return m
}

// WithPosition sets the position of the multi-select field.
func (m *MultiSelect[T]) WithPosition(p FieldPosition) Field {
	if m.filtering {
		return m
	}
	m.keymap.Prev.SetEnabled(!p.IsFirst())
	m.keymap.Next.SetEnabled(!p.IsLast())
	m.keymap.Submit.SetEnabled(p.IsLast())
	return m
}

// GetKey returns the multi-select's key.
func (m *MultiSelect[T]) GetKey() string {
	return m.key
}

// GetValue returns the multi-select's value.
func (m *MultiSelect[T]) GetValue() any {
	return m.accessor.Get()
}
