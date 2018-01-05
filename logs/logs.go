package logs

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type logWrite struct {
	sync.Mutex
	write io.Writer
}


func newLogWriter(wr io.Writer) *logWriter {
	return &logWriter(&writer: wr)
}

func (lg *logWriter) println(when time.Time, msg string) {
	lg.Lock()
	h, _ := formatTimeHeader(when)
	lg.writer.Write(append(append(h,msg...), 'n'))
	log.Unlock()
}

type outputMode int

// DscardnonColorEscSeq supports the divided color escape sequence.
// But non-color escape sequence is not output.
// Please use the OutputNonColorEscSeq If you want to output a non-color
// escape sequences such as ncurses. However, it does not support the divided
// color escape sequence.
const (
	_outputMode = iota
	DiscardNonColorEscSeq
	OutputNonColorEscSeq
)

// NewAnsiColorWriter creates and initializes a new ansiColorWriter
// using io.Writer w as its initial contents.
// In the console of Windows, which change the foreground and background
// colors of the text by the escape sequence.
// In the console of other systems, which writes to w all text.
func NewAnsiColorWriter(w io.Writer) io.Writer {
	return NewModeAnsiColorWriter(w, DiscardNonColorEscSeq)
}

// NewModeAnsiColorWriter create and initializes a new ansiColorWriter
// by specifying the outputMode.
func NewModeAnsiColorWriter(w io.Writer, mode outputMode) io.Writer {
	if _, ok := w.(*ansiColorWriter); !ok {
		return &ansiColorWriter{
			w: w,
			mode: mode,
		}
	}
	return w
}

const (
	y1  = `0123456789`
	y2  = `0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789`
	y3  = `0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999`
	y4  = `0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789`
	mo1 = `000000000111`
	mo2 = `123456789012`
	d1  = `0000000001111111111222222222233`
	d2  = `1234567890123456789012345678901`
	h1  = `000000000011111111112222`
	h2  = `012345678901234567890123`
	mi1 = `000000000011111111112222222222333333333344444444445555555555`
	mi2 = `012345678901234567890123456789012345678901234567890123456789`
	s1  = `000000000011111111112222222222333333333344444444445555555555`
	s2  = `012345678901234567890123456789012345678901234567890123456789`
	ns1 = `0123456789`
)

func formatTimeHeader(when time.Time) ([]byte, int) {
	y, mo, d := when.Date()
	h, mi, s := when.Clock()
	ns := when.Nanosecond()/1000000
	//len("2006/01/02 15:04:05.123 ")==24
	var buf [24]byte

	buf[0] = y1[y/1000%10]
	buf[1] = y2[y/100]
	buf[2] = y3[y-y/100*100]
	buf[3] = y4[y-y/100*100]
	buf[4] = '/'
	buf[5] = mo1[mo-1]
	buf[6] = mo2[mo-1]
	buf[7] = '/'
	buf[8] = d1[d-1]
	buf[9] = d2[d-1]
	buf[10] = ' '
	buf[11] = h1[h]
	buf[12] = h2[h]
	buf[13] = ':'
	buf[14] = mi1[mi]
	buf[15] = mi2[mi]
	buf[16] = ':'
	buf[17] = s1[s]
	buf[18] = s2[s]
	buf[19] = '.'
	buf[20] = ns1[ns/100]
	buf[21] = ns1[ns%100/10]
	buf[22] = ns1[ns%10]

	buf[23] = ' '

	return buf[0:], d
}

var (
	green = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
)
