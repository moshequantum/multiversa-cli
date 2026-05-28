package tui

import (
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/theme"
)

// Header renders the canonical Multiversa wizard header: an accented
// title, a dim subtitle, and an optional step-progress crumb of the
// form "step 2 of 5". The output never contains ANSI hyperlinks so
// it stays paste-safe in the user's terminal scrollback.
//
// Pass step=0,total=0 to omit the progress crumb (e.g. for one-shot
// commands like `detect`).
func Header(title, subtitle string, step, total int) string {
	var b strings.Builder
	b.WriteString(theme.Accent.Render(title))
	if step > 0 && total > 0 {
		b.WriteString("  ")
		b.WriteString(theme.Dim.Render(stepCrumb(step, total)))
	}
	b.WriteByte('\n')
	if subtitle != "" {
		b.WriteString(theme.Dim.Render(subtitle))
		b.WriteByte('\n')
	}
	return b.String()
}

// stepCrumb is split out so header_test can assert on its shape
// without coupling to the Header layout.
func stepCrumb(step, total int) string {
	return "step " + itoa(step) + " of " + itoa(total)
}

// itoa avoids strconv import inside this tiny file — keeps the
// header trivially auditable.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
