package strings

// The so-called spoken language join here works well for some Western
// languages. PRs for other languages are welcome, but do note that
// implementation for some languages will be less straightforward than the ones
// in use here.

import (
	"strings"
)

// Language is a spoken Language.
type Language int

// Available spoken lanaguges.
const (
	DE Language = iota
	DK
	EN
	ES
	FR
	IT
	NO
	PT
	SE
)

// String returns the English name of the [Language] code.
func (l Language) String() string {
	return map[Language]string{
		DE: "German",
		DK: "Danish",
		EN: "English",
		ES: "Spanish",
		FR: "French",
		IT: "Italian",
		NO: "Norwegian",
		PT: "Portuguese",
		SE: "Swedish",
	}[l]
}

func (l Language) conjuction() string {
	switch l {
	case DE:
		return "und"
	case DK:
		return "og"
	case EN:
		return "and"
	case ES:
		return "y"
	case FR:
		return "et"
	case NO:
		return "og"
	case IT:
		return "e"
	case PT:
		return "e"
	case SE:
		return "och"
	default:
		return ""
	}
}

func (l Language) separator() string {
	switch l {
	case DE, DK, EN, ES, FR, NO, IT, PT, SE:
		return ", "
	default:
		return " "
	}
}

// EnglishJoin joins a slice of strings with commas and the "and" conjugation
// before the final item. The Oxford comma can optionally be applied.
//
// Example:
//
//	str := EnglishJoin([]string{"meow", "purr", "raow"}, true)
//	fmt.Println(str) // meow, purr, and raow
func EnglishJoin(words []string, oxfordComma bool) string {
	return spokenLangJoin(words, EN, oxfordComma)
}

// SpokenLangaugeJoin joins a slice of strings with commas and a conjuction
// before the final item. You may specify the language with [Language].
//
// If you are using English and need the Oxford Comma, use [EnglishJoin].
//
// Example:
//
//	str := SpokenLanguageJoin([]string{"eins", "zwei", "drei"}, DE)
//	fmt.Println(str) // eins, zwei und drei
func SpokenLanguageJoin(words []string, language Language) string {
	return spokenLangJoin(words, language, false)
}

func spokenLangJoin(words []string, language Language, oxfordComma bool) string {
	conjuction := language.conjuction() + " "
	separator := language.separator()

	b := strings.Builder{}
	for i, word := range words {
		if word == "" {
			continue
		}

		if i == 0 {
			b.WriteString(word)
			continue
		}

		// Is this the final word?
		if len(words) > 1 && i == len(words)-1 {
			// Apply the Oxford comma if requested as long as the language is
			// English.
			if language == EN && oxfordComma && i > 1 {
				b.WriteString(separator)
			} else {
				b.WriteRune(' ')
			}

			b.WriteString(conjuction + word)
			continue
		}

		b.WriteString(separator + word)
	}
	return b.String()
}
