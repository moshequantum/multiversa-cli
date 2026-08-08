package steps

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moshequantum/multiversa-cli/internal/profile"
	"github.com/moshequantum/multiversa-cli/internal/theme"
)

const maxProjectOSNameLength = 64

var projectOSNamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9_-]*[A-Za-z0-9])?$`)

type Naming struct {
	DryRun        bool
	input         string
	validationErr string
	saveErr       string
	saving        bool
	width         int
}

type namingSavedMsg struct{ err error }

func NewNaming() Step {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	return newNamingWithSuggestion(hostname, username)
}

func newNamingWithSuggestion(hostname, username string) *Naming {
	suggestion := suggestProjectOSName(hostname, username)
	return &Naming{input: suggestion}
}

func (*Naming) Title() string { return "Nombre" }

func (*Naming) Init() tea.Cmd { return nil }

func (n *Naming) SetDryRun(dryRun bool) { n.DryRun = dryRun }

func (n *Naming) SetName(name string) {
	n.input = name
}

func (n *Naming) Name() string {
	return strings.TrimSpace(n.input)
}

func (n *Naming) Update(msg tea.Msg) (Step, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		n.width = message.Width
		return n, nil
	case namingSavedMsg:
		n.saving = false
		if message.err != nil {
			n.saveErr = "Corrige ~/.multiversa/profile.toml y vuelve a intentar: " + message.err.Error()
			return n, nil
		}
		n.saveErr = ""
		return n, Next
	case tea.KeyMsg:
		switch message.String() {
		case "esc":
			return n, Back
		case "enter":
			if n.saving {
				return n, nil
			}
			name := strings.TrimSpace(n.input)
			if err := validateProjectOSName(name); err != nil {
				n.validationErr = err.Error()
				return n, nil
			}
			n.SetName(name)
			n.validationErr = ""
			n.saveErr = ""
			if n.DryRun {
				return n, Next
			}
			n.saving = true
			return n, persistProjectOSName(name)
		case "backspace", "ctrl+h":
			n.input = trimLastRune(n.input)
			return n, nil
		case "ctrl+u":
			n.SetName("")
			return n, nil
		}
		if message.Type == tea.KeyRunes && len([]rune(n.input))+len(message.Runes) <= maxProjectOSNameLength {
			n.input += string(message.Runes)
		}
	}
	return n, nil
}

func (n *Naming) View() string {
	parts := []string{
		theme.Display.Render("¿Cómo se llamará tu ProjectOS?"),
		theme.Dim.Render("Usa un nombre único que puedan leer personas, agentes y manifiestos."),
		"",
		theme.Accent.Render("▸ ") + theme.Body.Render(n.input) + theme.Accent.Render("▌"),
	}
	if n.DryRun {
		parts = append(parts, "", theme.Dim.Render("dry-run · el nombre se mostrará sin escribir el perfil"))
	}
	if n.validationErr != "" {
		parts = append(parts, "", theme.Warn.Render(n.validationErr))
	}
	if n.saveErr != "" {
		parts = append(parts, "", theme.Warn.Render(n.saveErr))
	}
	if n.saving {
		parts = append(parts, "", theme.Dim.Render("Guardando nombre en ~/.multiversa/profile.toml…"))
	}
	parts = append(parts, "", theme.Dim.Render("[enter] continuar  ·  [ctrl-u] limpiar  ·  [esc] atrás"))
	return theme.Frame(n.width, lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func validateProjectOSName(name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return errors.New("Escribe un nombre para tu ProjectOS.")
	case len(name) > maxProjectOSNameLength:
		return fmt.Errorf("Usa un máximo de %d caracteres.", maxProjectOSNameLength)
	case !projectOSNamePattern.MatchString(name):
		return errors.New("Usa letras sin acentos, números, guiones o guion bajo; empieza y termina con letra o número.")
	}
	return nil
}

func suggestProjectOSName(hostname, username string) string {
	candidate := strings.TrimSpace(hostname)
	if candidate == "" || strings.EqualFold(candidate, "localhost") {
		candidate = strings.TrimSpace(username)
	}
	base := slugSafeNamePart(candidate)
	if base == "" {
		base = slugSafeNamePart(username)
	}
	if base == "" {
		base = "MiProject"
	}
	if len(base) > maxProjectOSNameLength-2 {
		base = strings.TrimRight(base[:maxProjectOSNameLength-2], "-_")
	}
	if !strings.HasSuffix(strings.ToLower(base), "os") {
		base += "OS"
	}
	return base
}

func slugSafeNamePart(value string) string {
	var out strings.Builder
	separatorPending := false
	for _, r := range value {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if separatorPending && out.Len() > 0 {
				out.WriteByte('-')
			}
			separatorPending = false
			out.WriteRune(r)
			continue
		}
		if r == '-' || r == '_' {
			if out.Len() > 0 {
				separatorPending = true
			}
			continue
		}
		if out.Len() > 0 {
			separatorPending = true
		}
	}
	return strings.Trim(out.String(), "-_")
}

func persistProjectOSName(name string) tea.Cmd {
	return func() tea.Msg {
		p, err := profile.Load()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return namingSavedMsg{err: err}
		}
		if !p.Level.IsValid() {
			p.Level = profile.Enthusiast
		}
		p.ProjectOSName = name
		return namingSavedMsg{err: p.Save()}
	}
}
