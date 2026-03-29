package output

import (
	"encoding/json"
	"fmt"
	"os"
)

var JSONMode bool

type Result struct {
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func PrintJSON(result Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func Step(current, total int, msg string) {
	if JSONMode {
		return
	}
	fmt.Printf("[%d/%d] %s\n", current, total, msg)
}

func OK(msg string) {
	if JSONMode {
		return
	}
	fmt.Printf("  [ok] %s\n", msg)
}

func Warn(msg string) {
	if JSONMode {
		return
	}
	fmt.Printf("  [warn] %s\n", msg)
}

func Err(msg string) {
	if JSONMode {
		return
	}
	fmt.Printf("  [err] %s\n", msg)
}

func Info(msg string) {
	if JSONMode {
		return
	}
	fmt.Println(msg)
}

func Infof(format string, args ...interface{}) {
	if JSONMode {
		return
	}
	fmt.Printf(format, args...)
}
