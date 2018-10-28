package hciframereceiver

import (
	"encoding/hex"
	"github.com/function61/gokit/assert"
	"github.com/function61/ruuvinator/pkg/ruuvinatortestdata"
	"github.com/function61/ruuvinator/pkg/utils"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	frames := []Frame{}

	err := ParseStream(strings.NewReader(ruuvinatortestdata.DemoStream), func(frame Frame) {
		frames = append(frames, frame)
	})

	assertOne := func(frame Frame, expectedDir HciDumpDirection, expectedData string) {
		t.Helper()

		dataAsHex := hex.EncodeToString(frame.Data)
		dataAsHexGroupedUppercased := strings.ToUpper(utils.SplitStringIntoGroupsOfTwo(dataAsHex, " "))

		assert.True(t, frame.Direction == expectedDir)
		assert.EqualString(t, dataAsHexGroupedUppercased, expectedData)
	}

	assert.True(t, err == nil)
	assert.True(t, len(frames) == 13)
	assertOne(frames[0], HciDumpDirectionInbound, "04 3E 1B 02 01 00 00 26 1B C6 08 03 60 0F 02 01 1A 0B FF 4C 00 09 06 03 15 C0 A8 0A 25 A8")
	assertOne(frames[12], HciDumpDirectionOutbound, "01 0C 20 02 00 00")
}
