package interact

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// PrintConfig holds configuration options for the print functions
type PrintConfig struct {
	// DisableColors disables ANSI color codes in output
	DisableColors bool
	// DisableEmojis disables emoji prefixes in output
	DisableEmojis bool
	// DisableBold disables bold formatting in output
	DisableBold bool
	// EnableDebug enables debug output
	EnableDebug bool
}

// DefaultConfig is the default print configuration
var DefaultConfig = PrintConfig{
	DisableColors: false,
	DisableEmojis: false,
	DisableBold:   false,
	EnableDebug:   false,
}

// ANSI color codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Purple     = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
)

// Emoji constants
const (
	InfoEmoji    = "ℹ️ "
	SuccessEmoji = "✅ "
	WarnEmoji    = "⚠️ "
	ErrorEmoji   = "❌ "
	DebugEmoji   = "🔍 "
)

// ConfigFromEnv creates a PrintConfig from environment variables
// NSX_NO_COLOR: Set to 1 to disable colors
// NSX_NO_EMOJI: Set to 1 to disable emojis
// NSX_NO_BOLD: Set to 1 to disable bold formatting
// NSX_DEBUG: Set to 1 to enable debug output
func ConfigFromEnv() PrintConfig {
	config := DefaultConfig

	if envVal, exists := os.LookupEnv("NSX_NO_COLOR"); exists {
		if val, err := strconv.ParseBool(envVal); err == nil {
			config.DisableColors = val
		} else if strings.ToLower(envVal) == "true" || envVal == "1" {
			config.DisableColors = true
		}
	}

	if envVal, exists := os.LookupEnv("NSX_NO_EMOJI"); exists {
		if val, err := strconv.ParseBool(envVal); err == nil {
			config.DisableEmojis = val
		} else if strings.ToLower(envVal) == "true" || envVal == "1" {
			config.DisableEmojis = true
		}
	}

	if envVal, exists := os.LookupEnv("NSX_NO_BOLD"); exists {
		if val, err := strconv.ParseBool(envVal); err == nil {
			config.DisableBold = val
		} else if strings.ToLower(envVal) == "true" || envVal == "1" {
			config.DisableBold = true
		}
	}

	if envVal, exists := os.LookupEnv("NSX_DEBUG"); exists {
		if val, err := strconv.ParseBool(envVal); err == nil {
			config.EnableDebug = val
		} else if strings.ToLower(envVal) == "true" || envVal == "1" {
			config.EnableDebug = true
		}
	}

	// Also respect NO_COLOR standard (https://no-color.org/)
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		config.DisableColors = true
	}

	return config
}

// ActiveConfig is the currently active print configuration
var ActiveConfig = ConfigFromEnv()

// SetConfig sets the active print configuration
func SetConfig(config PrintConfig) {
	ActiveConfig = config
}

// GetConfig returns the active print configuration
func GetConfig() PrintConfig {
	return ActiveConfig
}

// IsDebugEnabled returns whether debug output is enabled
func IsDebugEnabled() bool {
	return ActiveConfig.EnableDebug
}

// EnableDebug enables debug output
func EnableDebug() {
	config := ActiveConfig
	config.EnableDebug = true
	ActiveConfig = config
}

// DisableDebug disables debug output
func DisableDebug() {
	config := ActiveConfig
	config.EnableDebug = false
	ActiveConfig = config
}

// Info prints an informational message in blue with an info emoji
func Info(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(Blue, InfoEmoji, message, ActiveConfig)
}

// Success prints a success message in green with a success emoji
func Success(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(Green, SuccessEmoji, message, ActiveConfig)
}

// Warn prints a warning message in yellow with a warning emoji
func Warn(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(Yellow, WarnEmoji, message, ActiveConfig)
}

// Error prints an error message in red with an error emoji
func Error(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(Red, ErrorEmoji, message, ActiveConfig)
}

// Debug prints a debug message in cyan with a debug emoji, only if debug is enabled
func Debug(format string, a ...interface{}) {
	if !ActiveConfig.EnableDebug {
		return
	}
	message := fmt.Sprintf(format, a...)
	printWithConfig(Cyan, DebugEmoji, message, ActiveConfig)
}

// Print prints a regular message without color or emoji
func Print(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	fmt.Println(message)
}

// PrintWithColor prints a message with a specified color
func PrintWithColor(color string, format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(color, "", message, ActiveConfig)
}

// PrintWithEmoji prints a message with a specified emoji
func PrintWithEmoji(emoji string, format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig("", emoji, message, ActiveConfig)
}

// PrintWithColorAndEmoji prints a message with a specified color and emoji
func PrintWithColorAndEmoji(color string, emoji string, format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	printWithConfig(color, emoji, message, ActiveConfig)
}

// printWithConfig is a helper function that applies the configuration settings
func printWithConfig(color string, emoji string, message string, config PrintConfig) {
	var colorStart, colorEnd, boldStart, boldEnd, emojiPrefix string

	// Apply color if enabled
	if !config.DisableColors && color != "" {
		colorStart = color
		colorEnd = Reset
	}

	// Apply bold if enabled
	if !config.DisableColors && !config.DisableBold {
		boldStart = Bold
		boldEnd = ""
	}

	// Apply emoji if enabled
	if !config.DisableEmojis && emoji != "" {
		emojiPrefix = emoji
	}

	// Print the formatted message
	fmt.Printf("%s%s%s%s%s%s\n", boldStart, colorStart, emojiPrefix, message, colorEnd, boldEnd)
}
