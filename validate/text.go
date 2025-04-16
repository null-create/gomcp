package validate

import "fmt"

// DetectionCategory defines the type of problematic character found.
type DetectionCategory string

const (
	TagChar        DetectionCategory = "Unicode Tag (U+E0000-U+E007F)"
	BidiControl    DetectionCategory = "Bidirectional Control"
	DeprecatedChar DetectionCategory = "Deprecated/Non-Character"
	InvisibleFmt   DetectionCategory = "Invisible Formatting"
)

// DetectedCharInfo holds information about a detected problematic character.
type DetectedCharInfo struct {
	Rune       rune              `json:"rune"`
	Hex        string            `json:"hex"`   // Hex representation (e.g., "U+E0020")
	Index      int               `json:"index"` // Rune index (start byte) in the original string
	Category   DetectionCategory `json:"category"`
	Translated string            `json:"translated,omitempty"` // Plaintext equivalent, if applicable
}

// isTag checks if a rune is within the Unicode Tags block.
func isTag(r rune) bool {
	return r >= 0xE0000 && r <= 0xE007F
}

// isBidiControl checks for common bidirectional control characters.
// See: https://www.unicode.org/reports/tr9/
func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || // LRE, RLE, PDF, LRO, RLO
		(r >= 0x2066 && r <= 0x2069) || // LRI, RLI, FSI, PDI
		r == 0x061C // ALM (Arabic Letter Mark)
}

// isInvisibleFormatting checks for common invisible formatting chars.
// Note: Some might have legitimate uses (like ZWJ), but presence can be suspicious.
func isInvisibleFormatting(r rune) bool {
	switch r {
	case 0x200B, // Zero Width Space
		0x200C, // Zero Width Non-Joiner
		0x200D, // Zero Width Joiner
		0x2060, // Word Joiner
		0xFEFF: // Zero Width No-Break Space / BOM
		return true
	default:
		return false
	}
}

// isDeprecated checks for deprecated or explicitly non-character code points.
// This is less about hidden text and more about malformed/suspicious input.
func isDeprecated(r rune) bool {
	// Non-characters (e.g., U+FDD0-U+FDEF, U+nFFFE, U+nFFFF)
	if r >= 0xFDD0 && r <= 0xFDEF {
		return true
	}
	// Check for FFFE or FFFF at the end of any plane
	if (r&0xFFFE) == 0xFFFE || (r&0xFFFF) == 0xFFFF {
		return true
	}
	// Could add specific deprecated blocks if needed, e.g. U+2FF0..U+2FFF (Ideographic Description Characters)
	// but this might be too broad.
	return false
}

// DetectHiddenUnicode scans the input string for runes falling into
// predefined problematic categories like Unicode Tags, Bidi controls, etc.
// It returns a slice of DetectedCharInfo for each problematic rune found,
// including a translated representation where applicable.
func DetectHiddenUnicode(text string) []DetectedCharInfo {
	var detected = make([]DetectedCharInfo, 0)
	for index, r := range text {
		var category DetectionCategory
		var translated string // Variable to hold the translation
		isProblematic := false

		switch {
		case isTag(r):
			category = TagChar
			isProblematic = true
			// Attempt to translate Tag characters back to ASCII equivalents
			if r >= 0xE0020 && r <= 0xE007E {
				// Corresponds to ASCII printable characters U+0020 to U+007E
				translated = string(rune(r - 0xE0000))
			} else if r == 0xE007F {
				translated = "[Cancel Tag]" // Special tag
			} else if r == 0xE0001 {
				translated = "[Start Tag]" // Special tag
			}

		case isBidiControl(r):
			category = BidiControl
			isProblematic = true
			// Provide standard abbreviations for Bidi chars
			switch r {
			case 0x202A:
				translated = "[LRE]" // Left-to-Right Embedding
			case 0x202B:
				translated = "[RLE]" // Right-to-Left Embedding
			case 0x202C:
				translated = "[PDF]" // Pop Directional Formatting
			case 0x202D:
				translated = "[LRO]" // Left-to-Right Override
			case 0x202E:
				translated = "[RLO]" // Right-to-Left Override
			case 0x061C:
				translated = "[ALM]" // Arabic Letter Mark
			case 0x2066:
				translated = "[LRI]" // Left-to-Right Isolate
			case 0x2067:
				translated = "[RLI]" // Right-to-Left Isolate
			case 0x2068:
				translated = "[FSI]" // First Strong Isolate
			case 0x2069:
				translated = "[PDI]" // Pop Directional Isolate
			default:
				translated = "[Bidi]"
			}

		case isInvisibleFormatting(r):
			category = InvisibleFmt
			isProblematic = true
			// Provide names for invisible chars
			switch r {
			case 0x200B:
				translated = "[ZWSP]" // Zero Width Space
			case 0x200C:
				translated = "[ZWNJ]" // Zero Width Non-Joiner
			case 0x200D:
				translated = "[ZWJ]" // Zero Width Joiner
			case 0x2060:
				translated = "[WJ]" // Word Joiner
			case 0xFEFF:
				translated = "[ZWNBSP/BOM]" // Zero Width No-Break Space / Byte Order Mark
			default:
				translated = "[Invisible]"
			}

		case isDeprecated(r):
			category = DeprecatedChar
			isProblematic = true
			translated = "[Deprecated/NonChar]" // No direct translation
		}

		if isProblematic {
			detected = append(detected, DetectedCharInfo{
				Rune:       r,
				Hex:        fmt.Sprintf("U+%04X", r), // Format as Unicode hex
				Index:      index,                    // Note: This is byte index
				Category:   category,
				Translated: translated, // Assign the determined translation
			})
		}
	}
	return detected
}
