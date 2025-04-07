package stdio

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"

	msg "github.com/gomcp/types"
)

// STDIO Transport
func StartStdioTransport(handler msg.MessageHandler) {
	scanner := bufio.NewScanner(io.Reader(os.Stdin))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		handler(json.RawMessage(line))
	}
	if err := scanner.Err(); err != nil {
		log.Printf("STDIO scanner error: %v", err)
	}
}

func WriteStdioMessage(msg any) error {
	return json.NewEncoder(os.Stdout).Encode(msg)
}
