// Copyright 2017, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package memfile implements an in-memory emulation of os.File.
package memfile

import (
	"errors"
	"io"
	"sync"
)

var errInvalid = errors.New("invalid argument")

// File is an in-memory emulation of the I/O operations of os.File.
// The zero value for File is an empty file ready to use.
type File struct {
	m sync.Mutex
	b []byte
	i int
}

// New creates and initializes a new File using b as its initial contents.
// The new File takes ownership of b.
func New(b []byte) *File {
	return &File{b: b}
}

// Read reads up to len(b) bytes from the File.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns (0, io.EOF).
func (fb *File) Read(b []byte) (int, error) {
	fb.m.Lock()
	defer fb.m.Unlock()

	n, err := fb.readAt(b, int64(fb.i))
	fb.i += n
	return n, err
}

// ReadAt reads len(b) bytes from the File starting at byte offset.
// It returns the number of bytes read and the error, if any.
// At end of file, that error is io.EOF.
func (fb *File) ReadAt(b []byte, offset int64) (int, error) {
	fb.m.Lock()
	defer fb.m.Unlock()
	return fb.readAt(b, offset)
}
func (fb *File) readAt(b []byte, off int64) (int, error) {
	if off < 0 || int64(int(off)) < off {
		return 0, errInvalid
	}
	if off > int64(len(fb.b)) {
		return 0, io.EOF
	}
	n := copy(b, fb.b[off:])
	if n < len(b) {
		return n, io.EOF
	}
	return n, nil
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// If the current file offset is past the io.EOF, then the space in-between are
// implicitly filled with zero bytes.
func (fb *File) Write(b []byte) (int, error) {
	fb.m.Lock()
	defer fb.m.Unlock()

	n, err := fb.writeAt(b, int64(fb.i))
	fb.i += n
	return n, err
}

// WriteAt writes len(b) bytes to the File starting at byte offset.
// It returns the number of bytes written and an error, if any.
// If offset lies past io.EOF, then the space in-between are implicitly filled
// with zero bytes.
func (fb *File) WriteAt(b []byte, offset int64) (int, error) {
	fb.m.Lock()
	defer fb.m.Unlock()
	return fb.writeAt(b, offset)
}
func (fb *File) writeAt(b []byte, off int64) (int, error) {
	if off < 0 || int64(int(off)) < off {
		return 0, errInvalid
	}
	if off > int64(len(fb.b)) {
		fb.truncate(off)
	}
	n := copy(fb.b[off:], b)
	fb.b = append(fb.b, b[n:]...)
	return len(b), nil
}

// Seek sets the offset for the next Read or Write on file with offset,
// interpreted according to whence: 0 means relative to the origin of the file,
// 1 means relative to the current offset, and 2 means relative to the end.
func (fb *File) Seek(offset int64, whence int) (int64, error) {
	fb.m.Lock()
	defer fb.m.Unlock()

	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = int64(fb.i) + offset
	case io.SeekEnd:
		abs = int64(len(fb.b)) + offset
	default:
		return 0, errInvalid
	}
	if abs < 0 {
		return 0, errInvalid
	}
	fb.i = int(abs)
	return abs, nil
}

// Truncate changes the size of the file. It does not change the I/O offset.
func (fb *File) Truncate(n int64) error {
	fb.m.Lock()
	defer fb.m.Unlock()
	return fb.truncate(n)
}
func (fb *File) truncate(n int64) error {
	switch {
	case n < 0 || int64(int(n)) < n:
		return errInvalid
	case n <= int64(len(fb.b)):
		fb.b = fb.b[:n]
		return nil
	default:
		fb.b = append(fb.b, make([]byte, int(n)-len(fb.b))...)
		return nil
	}
}

// Bytes returns the full contents of the File.
// The result in only valid until the next Write, WriteAt, or Truncate call.
func (fb *File) Bytes() []byte {
	fb.m.Lock()
	defer fb.m.Unlock()
	return fb.b
}
