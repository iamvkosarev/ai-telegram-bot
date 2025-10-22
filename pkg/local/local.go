package local

import "fmt"

type Language string

const (
	Eng = Language("en")
	Rus = Language("ru")
)

type Localization struct {
	language Language
	text     string
}

type TextSet struct {
	Default          string
	translationsText map[Language]string
}

func NewTrans(language Language, text string) Localization {
	return Localization{
		language: language,
		text:     text,
	}
}

func NewSet(defaultText string, localizations ...Localization) TextSet {
	set := TextSet{
		Default:          defaultText,
		translationsText: make(map[Language]string),
	}
	for _, localization := range localizations {
		set.translationsText[localization.language] = localization.text
	}
	return set
}

func (l TextSet) Text(language Language) string {
	if text, ok := l.translationsText[language]; ok {
		return text
	}
	return l.Default
}

func (l TextSet) DefaultFormat(a ...any) string {
	return fmt.Sprintf(l.Default, a)
}

func (l TextSet) Format(language Language, a ...any) string {
	if text, ok := l.translationsText[language]; ok {
		return fmt.Sprintf(text, a)
	}
	return l.DefaultFormat(a)
}
