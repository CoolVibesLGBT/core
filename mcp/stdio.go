package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
)

func (r *MCPServer) ServeStdio(ctx context.Context, input io.Reader, output io.Writer) error {
	connection := r.NewConnection()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	writer := bufio.NewWriter(output)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var message JSONRPCMessage
		if err := json.Unmarshal(line, &message); err != nil {
			if err := writeStdioMessage(writer, NewJSONRPCErrorMessage(nil, ParseErrorCode, "parse error", map[string]any{"details": err.Error()})); err != nil {
				return err
			}
			continue
		}

		response := r.HandleMessage(ctx, connection, message)
		if response == nil {
			continue
		}

		if err := writeStdioMessage(writer, response); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func writeStdioMessage(writer *bufio.Writer, message *JSONRPCMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if _, err := writer.Write(append(payload, '\n')); err != nil {
		return err
	}

	return writer.Flush()
}
