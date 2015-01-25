// Copyright ©2012 The bíogo.bam Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sam

import (
	"bufio"
	"errors"
	"io"
)

type Reader struct {
	r *bufio.Reader
	h *Header
}

func NewReader(r io.Reader) (*Reader, error) {
	h, _ := NewHeader(nil, nil)
	sr := &Reader{
		r: bufio.NewReader(r),
		h: h,
	}

	var b []byte
	for {
		l, err := sr.r.ReadBytes('\n')
		if err != nil {
			return nil, io.ErrUnexpectedEOF
		}
		b = append(b, l...)
		p, err := sr.r.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if p[0] != '@' {
			break
		}
	}

	err := sr.h.UnmarshalText(b)
	if err != nil {
		return nil, err
	}

	return sr, nil
}

func (r *Reader) Header() *Header {
	return r.h
}

func (r *Reader) Read() (*Record, error) {
	b, err := r.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	b = b[:len(b)-1]
	if b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	var rec Record
	err = rec.UnmarshalSAM(r.h, b)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

type Writer struct {
	w     io.Writer
	flags int
}

func NewWriter(w io.Writer, h *Header, flags int) (*Writer, error) {
	if flags < FlagDecimal || flags > FlagString {
		return nil, errors.New("bam: flag format option out of range")
	}
	sw := &Writer{w: w, flags: flags}
	text, _ := h.MarshalText()
	_, err := w.Write(text)
	if err != nil {
		return nil, err
	}
	return sw, nil
}

func (w *Writer) Write(r *Record) error {
	b, err := r.MarshalSAM(w.flags)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.w.Write(b)
	return err
}

const (
	maxInt32 = int(int32(^uint32(0) >> 1))
	minInt32 = -int(maxInt32) - 1
)

const wordBits = 29

func validInt32(i int) bool    { return minInt32 <= i && i <= maxInt32 }
func validPos(i int) bool      { return 0 <= i && i <= 1<<wordBits-1 }
func validTmpltLen(i int) bool { return -(1<<wordBits) <= i && i <= 1<<wordBits-1 }
func validLen(i int) bool      { return 1 <= i && i <= 1<<wordBits-1 }

const nextBinShift = 3

const (
	level0 = uint16(((1 << (iota * nextBinShift)) - 1) / 7)
	level1
	level2
	level3
	level4
	level5
)

const (
	level0Shift = wordBits - (iota * nextBinShift)
	level1Shift
	level2Shift
	level3Shift
	level4Shift
	level5Shift
)

// calculate bin given an alignment covering [beg,end) (zero-based, half-close-half-open)
func reg2bin(beg, end int) uint16 {
	end--
	switch {
	case beg>>level5Shift == end>>level5Shift:
		return level5 + uint16(beg>>level5Shift)
	case beg>>level4Shift == end>>level4Shift:
		return level4 + uint16(beg>>level4Shift)
	case beg>>level3Shift == end>>level3Shift:
		return level3 + uint16(beg>>level3Shift)
	case beg>>level2Shift == end>>level2Shift:
		return level2 + uint16(beg>>level2Shift)
	case beg>>level1Shift == end>>level1Shift:
		return level1 + uint16(beg>>level1Shift)
	}
	return level0
}