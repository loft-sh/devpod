# Huh?

<p>
  <img alt="Hey there! I’m Glenn!" title="Hey there! I’m Glenn!" src="https://stuff.charm.sh/huh/glenn.png" width="400" />
  <br><br>
  <a href="https://github.com/charmbracelet/huh/releases"><img src="https://img.shields.io/github/release/charmbracelet/huh.svg" alt="Latest Release"></a>
  <a href="https://pkg.go.dev/github.com/charmbracelet/huh?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="Go Docs"></a>
  <a href="https://github.com/charmbracelet/huh/actions"><img src="https://github.com/charmbracelet/huh/actions/workflows/build.yml/badge.svg?branch=main" alt="Build Status"></a>
</p>

A simple, powerful library for building interactive forms and prompts in the terminal.

<img alt="Running a burger form" width="600" src="https://vhs.charm.sh/vhs-3J4i6HE3yBmz6SUO3HqILr.gif">

`huh?` is easy to use in a standalone fashion, can be
[integrated into a Bubble Tea application](#what-about-bubble-tea), and contains
a first-class [accessible mode](#accessibility) for screen readers.

The above example is running from a single Go program ([source](./examples/burger/main.go)).

## Tutorial

Let’s build a form for ordering burgers. To start, we’ll import the library and
define a few variables where’ll we store answers.

```go
package main

import "github.com/charmbracelet/huh"

var (
    burger       string
    toppings     []string
    sauceLevel   int
    name         string
    instructions string
    discount     bool
)
```

`huh?` separates forms into groups (you can think of groups as pages). Groups
are made of fields (e.g. `Select`, `Input`, `Text`). We will set up three
groups for the customer to fill out.

```go
form := huh.NewForm(
    huh.NewGroup(
        // Ask the user for a base burger and toppings.
        huh.NewSelect[string]().
            Title("Choose your burger").
            Options(
                huh.NewOption("Charmburger Classic", "classic"),
                huh.NewOption("Chickwich", "chickwich"),
                huh.NewOption("Fishburger", "fishburger"),
                huh.NewOption("Charmpossible™ Burger", "charmpossible"),
            ).
            Value(&burger), // store the chosen option in the "burger" variable

        // Let the user select multiple toppings.
        huh.NewMultiSelect[string]().
            Title("Toppings").
            Options(
                huh.NewOption("Lettuce", "lettuce").Selected(true),
                huh.NewOption("Tomatoes", "tomatoes").Selected(true),
                huh.NewOption("Jalapeños", "jalapeños"),
                huh.NewOption("Cheese", "cheese"),
                huh.NewOption("Vegan Cheese", "vegan cheese"),
                huh.NewOption("Nutella", "nutella"),
            ).
            Limit(4). // there’s a 4 topping limit!
            Value(&toppings),

        // Option values in selects and multi selects can be any type you
        // want. We’ve been recording strings above, but here we’ll store
        // answers as integers. Note the generic "[int]" directive below.
        huh.NewSelect[int]().
            Title("How much Charm Sauce do you want?").
            Options(
                huh.NewOption("None", 0),
                huh.NewOption("A little", 1),
                huh.NewOption("A lot", 2),
            ).
            Value(&sauceLevel),
    ),

    // Gather some final details about the order.
    huh.NewGroup(
        huh.NewInput().
            Title("What’s your name?").
            Value(&name).
            // Validating fields is easy. The form will mark erroneous fields
            // and display error messages accordingly.
            Validate(func(str string) error {
                if str == "Frank" {
                    return errors.New("Sorry, we don’t serve customers named Frank.")
                }
                return nil
            }),

        huh.NewText().
            Title("Special Instructions").
            CharLimit(400).
            Value(&instructions),

        huh.NewConfirm().
            Title("Would you like 15% off?").
            Value(&discount),
    ),
)
```

Finally, run the form:

```go
err := form.Run()
if err != nil {
    log.Fatal(err)
}

if !discount {
    fmt.Println("What? You didn’t take the discount?!")
}
```

And that’s it! For more info see [the full source][burgersource] for this
example as well as [the docs][docs].

If you need more dynamic forms that change based on input from previous fields,
check out the [dynamic forms](#dynamic-forms) example.

[burgersource]: ./examples/burger/main.go
[docs]: https://pkg.go.dev/github.com/charmbracelet/huh?tab=doc

## Field Reference

- [`Input`](#input): single line text input
- [`Text`](#text): multi-line text input
- [`Select`](#select): select an option from a list
- [`MultiSelect`](#multiple-select): select multiple options from a list
- [`Confirm`](#confirm): confirm an action (yes or no)

> [!TIP]
> Just want to prompt the user with a single field? Each field has a `Run`
> method that can be used as a shorthand for gathering quick and easy input.

```go
var name string

huh.NewInput().
    Title("What’s your name?").
    Value(&name).
    Run() // this is blocking...

fmt.Printf("Hey, %s!\n", name)
```

### Input

Prompt the user for a single line of text.

<img alt="Input field" width="600" src="https://vhs.charm.sh/vhs-1ULe9JbTHfwFmm3hweRVtD.gif">

```go
huh.NewInput().
    Title("What’s for lunch?").
    Prompt("?").
    Validate(isFood).
    Value(&lunch)
```

### Text

Prompt the user for multiple lines of text.

<img alt="Text field" width="600" src="https://vhs.charm.sh/vhs-2rrIuVSEf38bT0cwc8hfEG.gif">

```go
huh.NewText().
    Title("Tell me a story.").
    Validate(checkForPlagiarism).
    Value(&story)
```

### Select

Prompt the user to select a single option from a list.

<img alt="Select field" width="600" src="https://vhs.charm.sh/vhs-7wFqZlxMWgbWmOIpBqXJTi.gif">

```go
huh.NewSelect[string]().
    Title("Pick a country.").
    Options(
        huh.NewOption("United States", "US"),
        huh.NewOption("Germany", "DE"),
        huh.NewOption("Brazil", "BR"),
        huh.NewOption("Canada", "CA"),
    ).
    Value(&country)
```

### Multiple Select

Prompt the user to select multiple (zero or more) options from a list.

<img alt="Multiselect field" width="600" src="https://vhs.charm.sh/vhs-3TLImcoexOehRNLELysMpK.gif">

```go
huh.NewMultiSelect[string]().
    Options(
        huh.NewOption("Lettuce", "Lettuce").Selected(true),
        huh.NewOption("Tomatoes", "Tomatoes").Selected(true),
        huh.NewOption("Charm Sauce", "Charm Sauce"),
        huh.NewOption("Jalapeños", "Jalapeños"),
        huh.NewOption("Cheese", "Cheese"),
        huh.NewOption("Vegan Cheese", "Vegan Cheese"),
        huh.NewOption("Nutella", "Nutella"),
    ).
    Title("Toppings").
    Limit(4).
    Value(&toppings)
```

### Confirm

Prompt the user to confirm (Yes or No).

<img alt="Confirm field" width="600" src="https://vhs.charm.sh/vhs-2HeX5MdOxLsrWwsa0TNMIL.gif">

```go
huh.NewConfirm().
    Title("Are you sure?").
    Affirmative("Yes!").
    Negative("No.").
    Value(&confirm)
```

## Accessibility

`huh?` has a special rendering option designed specifically for screen readers.
You can enable it with `form.WithAccessible(true)`.

> [!TIP]
> We recommend setting this through an environment variable or configuration
> option to allow the user to control accessibility.

```go
accessibleMode := os.Getenv("ACCESSIBLE") != ""
form.WithAccessible(accessibleMode)
```

Accessible forms will drop TUIs in favor of standard prompts, providing better
dictation and feedback of the information on screen for the visually impaired.

<img alt="Accessible cuisine form" width="600" src="https://vhs.charm.sh/vhs-19xEBn4LgzPZDtgzXRRJYS.gif">

## Themes

`huh?` contains a powerful theme abstraction. Supply your own custom theme or
choose from one of the five predefined themes:

- `Charm`
- `Dracula`
- `Catppuccin`
- `Base 16`
- `Default`

<br />
<p>
    <img alt="Charm-themed form" width="400" src="https://stuff.charm.sh/huh/themes/charm-theme.png">
    <img alt="Dracula-themed form" width="400" src="https://stuff.charm.sh/huh/themes/dracula-theme.png">
    <img alt="Catppuccin-themed form" width="400" src="https://stuff.charm.sh/huh/themes/catppuccin-theme.png">
    <img alt="Base 16-themed form" width="400" src="https://stuff.charm.sh/huh/themes/basesixteen-theme.png">
    <img alt="Default-themed form" width="400" src="https://stuff.charm.sh/huh/themes/default-theme.png">
</p>

Themes can take advantage of the full range of
[Lip Gloss][lipgloss] style options. For a high level theme reference see
[the docs](https://pkg.go.dev/github.com/charmbracelet/huh#Theme).

[lipgloss]: https://github.com/charmbracelet/lipgloss

## Dynamic Forms

`huh?` forms can be as dynamic as your heart desires. Simply replace properties
with their equivalent `Func` to recompute the properties value every time a
different part of your form changes.

Here’s how you would build a simple country + state / province picker.

First, define some variables that we’ll use to store the user selection.

```go
var country string
var state string
```

Define your country select as you normally would:

```go
huh.NewSelect[string]().
    Options(huh.NewOptions("United States", "Canada", "Mexico")...).
    Value(&country).
    Title("Country").
```

Define your state select with `TitleFunc` and `OptionsFunc` instead of `Title`
and `Options`. This will allow you to change the title and options based on the
selection of the previous field, i.e. `country`.

To do this, we provide a `func() string` and a `binding any` to `TitleFunc`. The
function defines what to show for the title and the binding specifies what value
needs to change for the function to recompute. So if `country` changes (e.g. the
user changes the selection) we will recompute the function.

For `OptionsFunc`, we provide a `func() []Option[string]` and a `binding any`.
We’ll fetch the country’s states, provinces, or territories from an API. `huh`
will automatically handle caching for you.

> [!IMPORTANT]
> We have to pass `&country` as the binding to recompute the function only when
> `country` changes, otherwise we will hit the API too often.

```go
huh.NewSelect[string]().
    Value(&state).
    Height(8).
    TitleFunc(func() string {
        switch country {
        case "United States":
            return "State"
        case "Canada":
            return "Province"
        default:
            return "Territory"
        }
    }, &country).
    OptionsFunc(func() []huh.Option[string] {
        opts := fetchStatesForCountry(country)
        return huh.NewOptions(opts...)
    }, &country),
```

Lastly, run the `form` with these inputs.

```go
err := form.Run()
if err != nil {
    log.Fatal(err)
}
```

<img width="600" src="https://vhs.charm.sh/vhs-6FRmBjNi2aiRb4INPXwIjo.gif" alt="Country / State form with dynamic inputs running.">

## Bonus: Spinner

`huh?` ships with a standalone spinner package. It’s useful for indicating
background activity after a form is submitted.

<img alt="Spinner while making a burger" width="600" src="https://vhs.charm.sh/vhs-6HvYomAFP6H8mngOYWXvwJ.gif">

Create a new spinner, set a title, set the action (or provide a `Context`), and run the spinner:

<table>

<tr>
<td> <strong>Action Style</strong> </td><td> <strong>Context Style</strong> </td></tr>
<tr>
<td>

```go
err := spinner.New().
    Title("Making your burger...").
    Action(makeBurger).
    Run()

fmt.Println("Order up!")
```

</td>
<td>

```go
go makeBurger()

err := spinner.New().
    Type(spinner.Line).
    Title("Making your burger...").
    Context(ctx).
    Run()

fmt.Println("Order up!")
```

</td>
</tr>
</table>

For more on Spinners see the [spinner examples](./spinner/examples) and
[the spinner docs](https://pkg.go.dev/github.com/charmbracelet/huh/spinner).

## What about Bubble Tea?

<img alt="Bubbletea + Huh?" width="174" src="https://stuff.charm.sh/huh/bubbletea-huh.png">

In addition to its standalone mode, `huh?` has first-class support for
[Bubble Tea][tea] and can be easily integrated into Bubble Tea applications.
It’s incredibly useful in portions of your Bubble Tea application that need
form-like input.

<img alt="Bubble Tea embedded form example" width="800" src="https://vhs.charm.sh/vhs-3wGaB7EUKWmojeaHpARMUv.gif">

A `huh.Form` is merely a `tea.Model`, so you can use it just as
you would any other [Bubble](https://github.com/charmbracelet/bubbles).

```go
type Model struct {
    form *huh.Form // huh.Form is just a tea.Model
}

func NewModel() Model {
    return Model{
        form: huh.NewForm(
            huh.NewGroup(
                huh.NewSelect[string]().
                    Key("class").
                    Options(huh.NewOptions("Warrior", "Mage", "Rogue")...).
                    Title("Choose your class"),

            huh.NewSelect[int]().
                Key("level").
                Options(huh.NewOptions(1, 20, 9999)...).
                Title("Choose your level"),
            ),
        )
    }
}

func (m Model) Init() tea.Cmd {
    return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ...

    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }

    return m, cmd
}

func (m Model) View() string {
    if m.form.State == huh.StateCompleted {
        class := m.form.GetString("class")
        level := m.form.GetString("level")
        return fmt.Sprintf("You selected: %s, Lvl. %d", class, level)
    }
    return m.form.View()
}

```

For more info in using `huh?` in Bubble Tea applications see [the full Bubble
Tea example][example].

[tea]: https://github.com/charmbracelet/bubbletea
[bubbles]: https://github.com/charmbracelet/bubbles
[example]: https://github.com/charmbracelet/huh/blob/main/examples/bubbletea/main.go

## `Huh?` in the Wild
For some `Huh?` programs in production, see:

* [glyphs](https://github.com/maaslalani/glyphs): a unicode symbol picker
* [meteor](https://github.com/stefanlogue/meteor): a highly customisable conventional commit message tool
* [freeze](https://github.com/charmbracelet/freeze): a tool for generating images of code and terminal output
* [gum](https://github.com/charmbracelet/gum): a tool for glamorous shell scripts
* [savvy](https://github.com/getsavvyinc/savvy-cli): the easiest way to create, share, and run runbooks in the terminal


## Feedback

We’d love to hear your thoughts on this project. Feel free to drop us a note!

- [Twitter](https://twitter.com/charmcli)
- [The Fediverse](https://mastodon.social/@charmcli)
- [Discord](https://charm.sh/chat)

## Acknowledgments

`huh?` is inspired by the wonderful [Survey][survey] library by Alec Aivazis.

[survey]: https://github.com/AlecAivazis/survey

## License

[MIT](https://github.com/charmbracelet/bubbletea/raw/master/LICENSE)

---

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
