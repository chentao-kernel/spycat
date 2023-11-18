// This program demonstrates how to attach an eBPF program to a uretprobe.
// The program will be attached to the 'readline' symbol in the binary '/bin/bash' and print out
// the line which 'readline' functions returns to the caller.

//go:build amd64

package uprobe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"

	"github.com/chentao-kernel/spycat/internal"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"
)

// $BPF_CLANG and $BPF_CFLAGS are set by the Makefile.
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc $BPF_CLANG -cflags $BPF_CFLAGS -target amd64 -type event bpf uretprobe.c -- -I../headers

const (
	// The path to the ELF binary containing the function to trace.
	// On some distributions, the 'readline' function is provided by a
	// dynamically-linked library, so the path of the library will need
	// to be specified instead, e.g. /usr/lib/libreadline.so.8.
	// Use `ldd /bin/bash` to find these paths.
	binPath = "/bin/bash"
	symbol  = "readline"
)

type BpfSession struct {
	Session   *core.Session
	BpfProbes []io.Closer
	// internal field
	BpfObjs *bpfObjects
	// internal field
	Reader *core.BpfReader
}

func NewBpfSession(name string, config *core.SessionConfig) *BpfSession {
	return &BpfSession{
		Session: core.NewSession(name, config, 10e5),
		Reader:  &core.BpfReader{},
	}
}

func (b *BpfSession) Start() error {
	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
		return err
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("loading objects: %s", err)
		return err
	}
	defer objs.Close()

	b.BpfObjs = &objs
	// Open an ELF binary and read its symbols.
	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		log.Fatalf("opening executable: %s", err)
		return err
	}

	// Open a Uretprobe at the exit point of the symbol and attach
	// the pre-compiled eBPF program to it.
	up, err := ex.Uretprobe(symbol, objs.UretprobeBashReadline, nil)
	if err != nil {
		log.Fatalf("creating uretprobe: %s", err)
		return err
	}
	defer up.Close()

	// Open a perf event reader from userspace on the PERF_EVENT_ARRAY map
	// described in the eBPF C program.
	rd, err := perf.NewReader(objs.Events, os.Getpagesize())
	if err != nil {
		log.Fatalf("creating perf event reader: %s", err)
		return err
	}
	defer rd.Close()

	b.Reader.PerfReader = rd

	log.Println("Listening for events:", b.Session.Name())

	var event bpfEvent
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				return nil
			}
			log.Printf("reading from perf event reader: %s", err)
			continue
		}

		if record.LostSamples != 0 {
			log.Printf("perf event ring buffer full, dropped %d samples", record.LostSamples)
			continue
		}

		// Parse the perf event entry into a bpfEvent structure.
		// RawSample中是原始数据
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing perf event: %s", err)
			continue
		}
		b.ReadData(unix.ByteSliceToString(event.Line[:]))
		log.Printf("%s:%s return value: %s", binPath, symbol, unix.ByteSliceToString(event.Line[:]))
	}
	return nil
}

func (b *BpfSession) Stop() error {
	internal.CloseAll(b.BpfProbes...)
	internal.CloseAll(b.BpfObjs)
	if b.Reader.PerfReader != nil {
		b.Reader.PerfReader.Close()
	}
	if b.Reader.RingBufReader != nil {
		b.Reader.RingBufReader.Close()
	}
	return nil
}

func (b *BpfSession) ReadData(data any) error {
	b.Session.DataBuffer <- data
	return nil
}

func (b *BpfSession) ConsumeEvent() error {
	return nil
}

func (b *BpfSession) SendToReceiver() error {
	return nil
}

func (b *BpfSession) Name() string {
	return b.Session.Name()
}
