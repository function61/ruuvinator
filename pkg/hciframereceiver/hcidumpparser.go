package hciframereceiver

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

/*	There are three kind of lines in hcidump output:

	- Begins with "> "
	- Begins with "< " (we are not interested in this direction)
	- Begins with " " (is a continuation of a previous line)
*/
const (
	hciDumpPrefixInbound      = "> "
	hciDumpPrefixOutbound     = "< "
	hciDumpPrefixContinuation = "  "
)

func ParseStream(stream io.Reader, frameReceived func(Frame)) error {
	lineScanner := bufio.NewScanner(stream)
	lineScanner.Split(bufio.ScanLines)

	var currentDirection HciDumpDirection = 0
	currentLine := ""

	// since each line doesn't contain hint if the payload is continued in the next line,
	// only after we see that a new inbound/outbound msg encountered we know that previous
	// msg was finished as a whole
	emitPreviousFinishedLine := func() {
		if currentLine == "" {
			return
		}

		asBytes, err := hexStringToBytes(currentLine)
		if err != nil {
			panic(err)
		}

		frameReceived(Frame{
			Direction: currentDirection,
			Data:      asBytes,
		})

		currentLine = ""
	}

	for lineScanner.Scan() {
		line := lineScanner.Text()

		switch {
		case strings.HasPrefix(line, hciDumpPrefixInbound):
			emitPreviousFinishedLine()
			currentDirection = HciDumpDirectionInbound
			currentLine = line[len(hciDumpPrefixInbound):]
		case strings.HasPrefix(line, hciDumpPrefixOutbound):
			emitPreviousFinishedLine()
			currentDirection = HciDumpDirectionOutbound
			currentLine = line[len(hciDumpPrefixOutbound):]
		case strings.HasPrefix(line, hciDumpPrefixContinuation):
			currentLine += " " + line[len(hciDumpPrefixContinuation):]
		case strings.HasPrefix(line, "HCI sniffer"):
		case strings.HasPrefix(line, "device: hci"):
			continue // ignore useless crap that should've been in stderr
		default:
			return errors.New("invalid format for line: " + line)
		}
	}
	if err := lineScanner.Err(); err != nil {
		return err
	}

	// cannot emit here, since we don't know if the last line would have had continuation

	return nil
}

// input example: "FF 00 BA DF 00 D0"
func hexStringToBytes(hexStringWithSpaces string) ([]byte, error) {
	hexString := strings.Replace(hexStringWithSpaces, " ", "", -1)

	buf := &bytes.Buffer{}
	hexDecoder := hex.NewDecoder(strings.NewReader(hexString))
	if _, err := io.Copy(buf, hexDecoder); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
