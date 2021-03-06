package gmf

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

var (
	inputSampleFilename  string = "examples/tests-sample.mp4"
	outputSampleFilename string = "examples/tests-output.mp4"
	inputSampleWidth     int    = 640
	inputSampleHeight    int    = 480
)

func assert(i interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}

	return i
}

func TestCtxCreation(t *testing.T) {
	ctx := NewCtx()

	if ctx.avCtx == nil {
		t.Fatal("AVContext is not initialized")
	}

	ctx.Free()
}

func TestCtxInput(t *testing.T) {
	inputCtx, err := NewInputCtx(inputSampleFilename)
	if err != nil {
		t.Fatal(err)
	}

	inputCtx.CloseInput()
}

func TestCtxOutput(t *testing.T) {
	cases := map[interface{}]error{
		outputSampleFilename:                       nil,
		NewOutputFmt("mp4", "", ""):                nil,
		NewOutputFmt("", outputSampleFilename, ""): nil,
		NewOutputFmt("", "", "application/mp4"):    nil,
		NewOutputFmt("", "", "wrong/mime"):         errors.New(fmt.Sprintf("output format is not initialized. Unable to allocate context")),
	}

	for arg, expected := range cases {
		if outuptCtx, err := NewOutputCtx(arg); err != nil {
			if err.Error() != expected.Error() {
				t.Error("Unexpected error:", err)
			}
		} else {
			outuptCtx.CloseOutput()
		}
	}

	log.Println("OutputContext is OK.")
}

func TestCtxCloseEmpty(t *testing.T) {
	ctx := NewCtx()

	ctx.CloseInput()
	ctx.CloseOutput()
	ctx.Free()
}

func TestNewStream(t *testing.T) {
	ctx := NewCtx()
	if ctx.avCtx == nil {
		t.Fatal("AVContext is not initialized")
	}
	defer ctx.Free()

	c := assert(NewEncoder(AV_CODEC_ID_MPEG1VIDEO)).(*Codec)

	cc := NewCodecCtx(c)
	defer cc.Release()

	cc.SetTimeBase(AVR{1, 25})
	cc.SetDimension(320, 200)

	if ctx.IsGlobalHeader() {
		cc.SetFlag(CODEC_FLAG_GLOBAL_HEADER)
	}

	log.Println("Dummy stream is created")
}

func TestWriteHeader(t *testing.T) {
	outputCtx, err := NewOutputCtx(outputSampleFilename)
	if err != nil {
		t.Fatal(err)
	}

	// write_header needs a valid stream with code context initialized
	c := assert(NewEncoder(AV_CODEC_ID_MPEG1VIDEO)).(*Codec)
	outputCtx.NewStream(c).SetCodecCtx(NewCodecCtx(c).SetTimeBase(AVR{1, 25}).SetDimension(10, 10).SetFlag(CODEC_FLAG_GLOBAL_HEADER))

	if err := outputCtx.WriteHeader(); err != nil {
		t.Fatal(err)
	}

	log.Println("Header has been written to", outputSampleFilename)

	if err := os.Remove(outputSampleFilename); err != nil {
		log.Fatal(err)
	}
}

func TestPacketsIterator(t *testing.T) {
	inputCtx, err := NewInputCtx(inputSampleFilename)
	if err != nil {
		t.Fatal(err)
	}

	defer inputCtx.CloseInput()

	for packet := range inputCtx.Packets() {
		if packet.Size() <= 0 {
			t.Fatal("Expected size > 0")
		} else {
			log.Printf("One packet has been read. size: %v, pts: %v\n", packet.Size(), packet.Pts())
		}

		break
	}
}

var section *io.SectionReader

func customReader() ([]byte, int) {
	var file *os.File
	var err error

	if section == nil {
		file, err = os.Open("tmp/ref.mp4")
		if err != nil {
			panic(err)
		}

		fi, err := file.Stat()
		if err != nil {
			panic(err)
		}

		section = io.NewSectionReader(file, 0, fi.Size())
	}

	b := make([]byte, IO_BUFFER_SIZE)

	n, err := section.Read(b)
	if err != nil {
		fmt.Println("section.Read():", err)
		file.Close()
	}

	return b, n
}

func TestAVIOContext(t *testing.T) {
	ictx := NewCtx()

	if err := ictx.SetInputFormat("mov"); err != nil {
		t.Fatal(err)
	}

	avioCtx, err := NewAVIOContext(ictx, &AVIOHandlers{ReadPacket: customReader})
	if err != nil {
		t.Fatal(err)
	}

	ictx.SetPb(avioCtx).SetFlag(AV_NOPTS_VALUE).OpenInput("")

	for p := range ictx.Packets() {
		_ = p
	}

	ictx.CloseInput()
}

func ExampleNewAVIOContext(t *testing.T) {
	ctx := NewCtx()

	// In this example, we're using custom reader implementation,
	// so we should specify format manually.
	if err := ctx.SetInputFormat("mov"); err != nil {
		t.Fatal(err)
	}

	avioCtx, err := NewAVIOContext(ctx, &AVIOHandlers{ReadPacket: customReader})
	if err != nil {
		t.Fatal(err)
	}

	// Setting up AVFormatContext.pb
	ctx.SetPb(avioCtx).SetFlag(AV_NOPTS_VALUE)

	// Calling OpenInput with empty arg, because all files stuff we're doing in custom reader.
	// But the library have to initialize some stuff, so we call it anyway.
	ctx.OpenInput("")

	for p := range ctx.Packets() {
		_ = p
	}
}
