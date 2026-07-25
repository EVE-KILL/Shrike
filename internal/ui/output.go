package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Success, Warn and Error are the three verdict lines. Success goes to stdout
// because it is part of a command's output; Warn and Error go to stderr so a
// caller piping stdout into jq still sees them.
func Success(format string, args ...any) {
	if JSONMode {
		return
	}
	mark := "[OK]"
	if ColorEnabled {
		mark = StyleSuccess.Render("✓")
	}
	fmt.Fprintf(os.Stdout, "%s %s\n", mark, fmt.Sprintf(format, args...))
}

func Warn(format string, args ...any) {
	mark := "[WARN]"
	if ColorEnabled {
		mark = StyleWarn.Render("!")
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", mark, fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	mark := "[ERROR]"
	if ColorEnabled {
		mark = StyleError.Render("✗")
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", mark, fmt.Sprintf(format, args...))
}

// Section prints a group heading. Suppressed in JSON mode along with every
// other decoration.
func Section(title string) {
	if JSONMode {
		return
	}
	if ColorEnabled {
		fmt.Printf("\n%s\n", StyleSection.Render(title))
		return
	}
	fmt.Printf("\n%s\n", strings.ToUpper(title))
}

// KV prints an aligned label/value pair. labelWidth is fixed rather than
// measured so successive calls align without the caller pre-computing a max.
const labelWidth = 18

func KV(label, value string) {
	if JSONMode {
		return
	}
	padded := fmt.Sprintf("%-*s", labelWidth, label+":")
	if ColorEnabled {
		fmt.Printf("  %s %s\n", StyleDim.Render(padded), value)
		return
	}
	fmt.Printf("  %s %s\n", padded, value)
}

func Println(args ...any) {
	if JSONMode {
		return
	}
	fmt.Println(args...)
}

func Printf(format string, args ...any) {
	if JSONMode {
		return
	}
	fmt.Printf(format, args...)
}

func Newline() {
	if JSONMode {
		return
	}
	fmt.Println()
}

// JSON writes v as indented JSON to stdout. This is the one output path that
// deliberately ignores JSONMode — it *is* the JSON mode payload.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
