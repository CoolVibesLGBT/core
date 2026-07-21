package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
)

type stdioScanResult struct {
	line []byte
	err  error
}

func (r *MCPServer) ServeStdio(ctx context.Context, input io.Reader, output io.Writer) error {
	connection := r.NewConnection()

	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	writer := bufio.NewWriter(output)

	scanResults := make(chan stdioScanResult)
	go func() {
		defer close(scanResults)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case scanResults <- stdioScanResult{line: line}:
			case <-ctx.Done():
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case scanResults <- stdioScanResult{err: err}:
			case <-ctx.Done():
			}
		}
	}()

	for {
		var scanResult stdioScanResult
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result, ok := <-scanResults:
			if !ok {
				return nil
			}
			scanResult = result
		}
		if scanResult.err != nil {
			return scanResult.err
		}

		line := scanResult.line
		if len(line) == 0 {
			continue
		}

		messages, isBatch, err := DecodeWireMessages(line)
		if err != nil {
			code := ParseErrorCode
			message := "parse error"
			if errors.Is(err, ErrEmptyBatch) {
				code = InvalidRequestCode
				message = "invalid request"
			}
			if err := writeStdioMessages(writer, []*JSONRPCMessage{
				NewJSONRPCErrorMessage(nil, code, message, map[string]any{"details": err.Error()}),
			}, false); err != nil {
				return err
			}
			continue
		}

		if isBatch {
			state := connection.Snapshot()
			if !state.Initialized || !r.SupportsBatch(state.ProtocolVersion) {
				if err := writeStdioMessages(writer, []*JSONRPCMessage{
					NewJSONRPCErrorMessage(nil, InvalidRequestCode, "JSON-RPC batches are only supported for initialized sessions using protocol version 2025-03-26", nil),
				}, false); err != nil {
					return err
				}
				continue
			}
		}

		responses := r.HandleMessages(ctx, connection, messages)
		if len(responses) == 0 {
			continue
		}

		if err := writeStdioMessages(writer, responses, isBatch); err != nil {
			return err
		}
	}
}

func writeStdioMessages(writer *bufio.Writer, messages []*JSONRPCMessage, isBatch bool) error {
	var (
		payload []byte
		err     error
	)

	if isBatch {
		payload, err = json.Marshal(messages)
	} else {
		payload, err = json.Marshal(messages[0])
	}
	if err != nil {
		return err
	}

	if _, err := writer.Write(append(payload, '\n')); err != nil {
		return err
	}

	return writer.Flush()
}
