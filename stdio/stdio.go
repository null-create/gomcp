package stdio

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
)

type MessageHandler func(message json.RawMessage)

// STDIO Transport
func StartStdioTransport(handler MessageHandler) {
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

func WriteStdioMessage(msg interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(msg)
}
