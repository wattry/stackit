package common

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadMessage resolves a commit message from either an inline `-m` string or
// a `--message-file` path. If messageFile is "-", the message is read from
// stdin. Trailing whitespace is trimmed so `echo "msg" | stackit create -F -`
// behaves the same as `stackit create -m "msg"`.
//
// Returns an error if both message and messageFile are non-empty.
func ReadMessage(message, messageFile string) (string, error) {
	if message != "" && messageFile != "" {
		return "", fmt.Errorf("--message and --message-file are mutually exclusive")
	}

	if messageFile == "" {
		return message, nil
	}

	return readMessageFrom(messageFile, os.Stdin)
}

func readMessageFrom(path string, stdin io.Reader) (string, error) {
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read message file: %w", err)
	}
	return strings.TrimRight(string(data), "\n\r\t "), nil
}
