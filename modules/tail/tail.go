package Utf16tail

import (
	"bufio"
	"bytes"
	"errors"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"os"
	"time"
)

// Line text structure
type Line struct {
	Text string
	Err  error
}

// UTF16 compatible tail structure
type Tail struct {
	FileName string
	Lines    chan *Line
	stop     chan struct{}
}

// Stop tail channel
func (t *Tail) Stop() {
	close(t.stop)
}

// scanCRLF costomization split character
func scanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			end := i
			if i > 0 && data[i-1] == '\r' {
				end--
			}
			return i + 1, bytes.TrimRight(data[:end], "\r"), nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), bytes.TrimRight(data, "\r"), nil
	}
	return 0, nil, nil
}

// decodeUTF16Lines recoding data from UTF-16LE to UTF-8
func decodeUTF16Lines(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var decoder *encoding.Decoder
	if len(data) >= 2 {
		if bytes.Equal(data[:2], []byte{0xFF, 0xFE}) {
			decoder = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
			data = data[2:]
		} else if bytes.Equal(data[:2], []byte{0xFE, 0xFF}) {
			decoder = unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()
			data = data[2:]
		} else {
			decoder = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		}
	} else {
		return nil, errors.New("invalid UTF-16 data")
	}
	reader := transform.NewReader(bytes.NewReader(data), decoder)
	buf := bufio.NewScanner(reader)
	buf.Split(scanCRLF)

	var lines []string
	for buf.Scan() {
		lines = append(lines, buf.Text())
	}
	if err := buf.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

// watching file update and read new text
func (t *Tail) watch() {
	var offset int64 = 0
	if fi, err := os.Stat(t.FileName); err == nil {
		offset = fi.Size()
	}
	for {
		select {
		case <-t.stop:
			close(t.Lines)
			return
		default:
			fi, err := os.Stat(t.FileName)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			size := fi.Size()
			if size < offset {
				//file might be regenerated
				offset = 0
			} else if size > offset {
				//new contents appear
				f, err := os.Open(t.FileName)
				if err != nil {
					t.Lines <- &Line{Err: err}
					time.Sleep(time.Second)
					continue
				}
				f.Seek(offset, io.SeekStart)
				data, _ := io.ReadAll(f)
				f.Close()

				lines, err := decodeUTF16Lines(data)
				if err != nil {
					t.Lines <- &Line{Err: err}
				}
				for _, l := range lines {
					if l != "" {
						t.Lines <- &Line{Text: l}
					}
				}
				offset = size
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Create a Tail instance (common to tail -f)
func New(filename string) *Tail {
	t := &Tail{
		FileName: filename,
		Lines:    make(chan *Line, 100),
		stop:     make(chan struct{}),
	}

	go t.watch()
	return t
}
