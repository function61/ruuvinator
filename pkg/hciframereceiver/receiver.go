package hciframereceiver

import (
	"context"
	"fmt"
	"github.com/function61/gokit/logger"
	"github.com/function61/gokit/stopper"
	"os"
	"os/exec"
)

type Frame struct {
	Direction HciDumpDirection
	Data      []byte
}

type HciDumpDirection int

const (
	HciDumpDirectionInbound HciDumpDirection = iota
	HciDumpDirectionOutbound
)

func Run(ctx context.Context, frameReceived func(Frame)) {
	workers := stopper.NewManager()

	go leScanner(ctx, workers.Stopper())
	go hciDumper(ctx, frameReceived, workers.Stopper())

	<-ctx.Done()

	workers.StopAllWorkersAndWait()
}

func leScanner(ctx context.Context, stop *stopper.Stopper) {
	defer stop.Done()

	log := logger.New("lescan")
	log.Info("starting")
	defer log.Info("stopped")

	leScan := exec.CommandContext(ctx, "hcitool", "lescan", "--duplicates", "--passive")
	leScan.Stderr = os.Stderr
	if err := leScan.Run(); err != nil {
		log.Error(err.Error())
	}
}

func hciDumper(ctx context.Context, frameReceived func(Frame), stop *stopper.Stopper) {
	defer stop.Done()

	log := logger.New("hcidump")
	log.Info("starting")
	defer log.Info("stopped")

	hciDumper := exec.CommandContext(ctx, "hcidump", "--raw")
	hciDumper.Stderr = os.Stderr
	hciDumperOutput, err := hciDumper.StdoutPipe()
	if err != nil {
		panic(err)
	}

	go func() {
		// write hciDumperOutput to parser which will invoke frameReceived
		// for each received frame
		if err := ParseStream(hciDumperOutput, frameReceived); err != nil {
			log.Error(fmt.Sprintf("hcidumpOutputParser: %s", err))
		}
	}()

	if err := hciDumper.Run(); err != nil {
		log.Error(err.Error())
	}
}
