package stdio

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	msg "github.com/gomcp/types"
)

// STDIO Transport. Pass a messageHandler for I/O processing.
func StartStdioTransport(handler msg.MessageHandler) error {
	scanner := bufio.NewScanner(io.Reader(os.Stdin))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := handler(json.RawMessage(line)); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
