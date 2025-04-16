package validate

import (
	"reflect" // Used for DeepEqual comparison of slices
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectHiddenUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []DetectedCharInfo // Expected slice of detected issues
	}{
		{
			name:     "Empty String",
			input:    "",
			expected: []DetectedCharInfo{}, // Expect empty slice
		},
		{
			name:     "Clean ASCII String",
			input:    "Hello, world!",
			expected: []DetectedCharInfo{},
		},
		{
			name:     "Clean Multi-byte String (Japanese)",
			input:    "こんにちは世界", // Konnichiwa Sekai
			expected: []DetectedCharInfo{},
		},
		{
			name:     "Clean Multi-byte String (Emoji)",
			input:    "Test with 😊 emoji",
			expected: []DetectedCharInfo{},
		},
		{
			name:  "Single Tag Character (Printable)",
			input: "A\U000E0042C", // Embeds Tag 'B' (U+E0042)
			expected: []DetectedCharInfo{
				{Rune: '\U000E0042', Hex: "U+E0042", Index: 1, Category: TagChar, Translated: "B"},
			},
		},
		{
			name:  "Single Tag Character (Cancel Tag)",
			input: "End\U000E007F",
			expected: []DetectedCharInfo{
				{Rune: '\U000E007F', Hex: "U+E007F", Index: 3, Category: TagChar, Translated: "[Cancel Tag]"},
			},
		},
		{
			name:  "Single Tag Character (Other Control - No Translation)",
			input: "Nul\U000E0000Char", // U+E0000
			expected: []DetectedCharInfo{
				// Note: Translated is empty/omitted based on current logic
				{Rune: '\U000E0000', Hex: "U+E0000", Index: 3, Category: TagChar, Translated: ""},
			},
		},
		{
			name: "Full Tag Sequence Example",
			// "SST This is another test payload CT" using tags
			input: "\U000E0001\U000E0054\U000E0068\U000E0069\U000E0073\U000E0020\U000E0069\U000E0073\U000E0020\U000E0061\U000E006E\U000E006F\U000E0074\U000E0068\U000E0065\U000E0072\U000E0020\U000E0074\U000E0065\U000E0073\U000E0074\U000E0020\U000E0070\U000E0061\U000E0079\U000E006C\U000E006F\U000E0061\U000E0064\U000E007F",
			expected: []DetectedCharInfo{
				{Rune: '\U000E0001', Hex: "U+E0001", Index: 0, Category: TagChar, Translated: "[Start Tag]"}, // Assuming [Start Tag] translation was added
				{Rune: '\U000E0054', Hex: "U+E0054", Index: 4, Category: TagChar, Translated: "T"},
				{Rune: '\U000E0068', Hex: "U+E0068", Index: 8, Category: TagChar, Translated: "h"},
				{Rune: '\U000E0069', Hex: "U+E0069", Index: 12, Category: TagChar, Translated: "i"},
				{Rune: '\U000E0073', Hex: "U+E0073", Index: 16, Category: TagChar, Translated: "s"},
				{Rune: '\U000E0020', Hex: "U+E0020", Index: 20, Category: TagChar, Translated: " "},
				{Rune: '\U000E0069', Hex: "U+E0069", Index: 24, Category: TagChar, Translated: "i"},
				{Rune: '\U000E0073', Hex: "U+E0073", Index: 28, Category: TagChar, Translated: "s"},
				{Rune: '\U000E0020', Hex: "U+E0020", Index: 32, Category: TagChar, Translated: " "},
				{Rune: '\U000E0061', Hex: "U+E0061", Index: 36, Category: TagChar, Translated: "a"},
				{Rune: '\U000E006E', Hex: "U+E006E", Index: 40, Category: TagChar, Translated: "n"},
				{Rune: '\U000E006F', Hex: "U+E006F", Index: 44, Category: TagChar, Translated: "o"},
				{Rune: '\U000E0074', Hex: "U+E0074", Index: 48, Category: TagChar, Translated: "t"},
				{Rune: '\U000E0068', Hex: "U+E0068", Index: 52, Category: TagChar, Translated: "h"},
				{Rune: '\U000E0065', Hex: "U+E0065", Index: 56, Category: TagChar, Translated: "e"},
				{Rune: '\U000E0072', Hex: "U+E0072", Index: 60, Category: TagChar, Translated: "r"},
				{Rune: '\U000E0020', Hex: "U+E0020", Index: 64, Category: TagChar, Translated: " "},
				{Rune: '\U000E0074', Hex: "U+E0074", Index: 68, Category: TagChar, Translated: "t"},
				{Rune: '\U000E0065', Hex: "U+E0065", Index: 72, Category: TagChar, Translated: "e"},
				{Rune: '\U000E0073', Hex: "U+E0073", Index: 76, Category: TagChar, Translated: "s"},
				{Rune: '\U000E0074', Hex: "U+E0074", Index: 80, Category: TagChar, Translated: "t"},
				{Rune: '\U000E0020', Hex: "U+E0020", Index: 84, Category: TagChar, Translated: " "},
				{Rune: '\U000E0070', Hex: "U+E0070", Index: 88, Category: TagChar, Translated: "p"},
				{Rune: '\U000E0061', Hex: "U+E0061", Index: 92, Category: TagChar, Translated: "a"},
				{Rune: '\U000E0079', Hex: "U+E0079", Index: 96, Category: TagChar, Translated: "y"},
				{Rune: '\U000E006C', Hex: "U+E006C", Index: 100, Category: TagChar, Translated: "l"},
				{Rune: '\U000E006F', Hex: "U+E006F", Index: 104, Category: TagChar, Translated: "o"},
				{Rune: '\U000E0061', Hex: "U+E0061", Index: 108, Category: TagChar, Translated: "a"},
				{Rune: '\U000E0064', Hex: "U+E0064", Index: 112, Category: TagChar, Translated: "d"},
				{Rune: '\U000E007F', Hex: "U+E007F", Index: 116, Category: TagChar, Translated: "[Cancel Tag]"},
			},
		},
		{
			name:  "Bidi Control Characters",
			input: "Hello\u202EWRLD", // U+202E is RLO
			expected: []DetectedCharInfo{
				{Rune: '\u202E', Hex: "U+202E", Index: 5, Category: BidiControl, Translated: "[RLO]"},
			},
		},
		{
			name:  "Invisible Formatting Characters",
			input: "Click\u200BHere", // U+200B is ZWSP
			expected: []DetectedCharInfo{
				{Rune: '\u200B', Hex: "U+200B", Index: 5, Category: InvisibleFmt, Translated: "[ZWSP]"},
			},
		},
		{
			name:  "Deprecated/Non-Character",
			input: "Invalid\uFDD0Char", // U+FDD0 is a non-character
			expected: []DetectedCharInfo{
				{Rune: '\uFDD0', Hex: "U+FDD0", Index: 7, Category: DeprecatedChar, Translated: "[Deprecated/NonChar]"},
			},
		},
		{
			name:  "Mixed Problematic Characters",
			input: "Command: \U000E0072\U000E006D\u202Etxt.evil\U000E007F", // rm<RLO>txt.evil<CancelTag>
			expected: []DetectedCharInfo{
				{Rune: '\U000E0072', Hex: "U+E0072", Index: 9, Category: TagChar, Translated: "r"},
				{Rune: '\U000E006D', Hex: "U+E006D", Index: 13, Category: TagChar, Translated: "m"},
				{Rune: '\u202E', Hex: "U+202E", Index: 17, Category: BidiControl, Translated: "[RLO]"},         // Note index carefully
				{Rune: '\U000E007F', Hex: "U+E007F", Index: 28, Category: TagChar, Translated: "[Cancel Tag]"}, // Index after RLO and txt.evil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualDetections := DetectHiddenUnicode(tc.input)

			// Use require for length check, as mismatch makes element checks pointless
			require.Len(t, actualDetections, len(tc.expected), "Number of detections mismatch")

			// Compare element by element for detailed checking
			if !reflect.DeepEqual(tc.expected, actualDetections) {
				assert.Equal(t, tc.expected, actualDetections, "Detected items mismatch")
			}

			// Alternatively, loop and assert each element if more granular checks needed:
			// for i := range tc.expected {
			//     assert.Equal(t, tc.expected[i].Rune, actualDetections[i].Rune, "Rune mismatch at index %d", i)
			//     assert.Equal(t, tc.expected[i].Hex, actualDetections[i].Hex, "Hex mismatch at index %d", i)
			//     assert.Equal(t, tc.expected[i].Index, actualDetections[i].Index, "Index mismatch at index %d", i)
			//     assert.Equal(t, tc.expected[i].Category, actualDetections[i].Category, "Category mismatch at index %d", i)
			//     assert.Equal(t, tc.expected[i].Translated, actualDetections[i].Translated, "Translated mismatch at index %d", i)
			// }
		})
	}
}
