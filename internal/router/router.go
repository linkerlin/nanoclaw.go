package router

import (
	"fmt"
	"strings"

	"github.com/linkerlin/nanoclaw.go/internal/types"
)

// EscapeXML replaces XML special characters in s.
func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// FormatMessages converts a slice of messages into the XML-like prompt block
// that the agent expects.
func FormatMessages(messages []types.NewMessage) string {
	if len(messages) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<messages>\n")
	for _, m := range messages {
		role := "user"
		if m.IsBotMessage {
			role = "assistant"
		}
		fmt.Fprintf(&sb, "  <message role=%q sender=%q timestamp=%q>\n    %s\n  </message>\n",
			role, EscapeXML(m.SenderName), EscapeXML(m.Timestamp), EscapeXML(m.Content))
	}
	sb.WriteString("</messages>")
	return sb.String()
}

// FormatOutbound cleans up the raw agent response for display.
func FormatOutbound(rawText string) string {
	return strings.TrimSpace(rawText)
}
