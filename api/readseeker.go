package api

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	_BUFFERING_AMOUNT = 256
	_BUFFERING_PERIOD = 100 * time.Millisecond
)

var errOutOfSize = errors.New("position is out of data size")

type HttpReadSeeker struct {
	source      io.ReadCloser
	bufferTimer *time.Ticker
	closed      chan bool
	readBuffer  []byte
	readIndex   int64
	totalSize   int64
	buffered    bool
	done        bool
	mux         sync.Mutex
}

func newReadSeaker(rc io.ReadCloser, totalSize int64) *HttpReadSeeker {
	rs := HttpReadSeeker{
		source:      rc,
		totalSize:   totalSize,
		bufferTimer: time.NewTicker(_BUFFERING_PERIOD),
		closed:      make(chan bool),
	}

	go rs.bufferFrames(_BUFFERING_AMOUNT)
	return &rs
}

func (h *HttpReadSeeker) bufferFrames(size int64) {
	for {
		h.mux.Lock()

		if h.buffered || h.totalSize <= int64(len(h.readBuffer)) {
			h.buffered = true
			h.mux.Unlock()
			return
		}

		buf := make([]byte, size)
		n, err := h.source.Read(buf)
		if err == nil || err == io.EOF {
			h.readBuffer = append(h.readBuffer, buf[:n]...)
			if err == io.EOF {
				h.buffered = true
				h.mux.Unlock()
				return
			}
		}

		h.mux.Unlock()

		// await next Read call or timer expiration
		select {
		case <-h.bufferTimer.C:
			continue
		case <-h.closed:
			return
		}
	}
}

func (h *HttpReadSeeker) Close() error {
	var err error

	h.mux.Lock()
	defer h.mux.Unlock()

	h.readBuffer = nil
	if !h.buffered {
		h.bufferTimer.Stop()
		h.buffered = true
		close(h.closed)
	}

	if !h.done {
		err = h.source.Close()
		h.done = true
	}

	return err
}

func (h *HttpReadSeeker) Length() int64 {
	return int64(h.totalSize)
}

func (h *HttpReadSeeker) Read(dest []byte) (n int, err error) {
	h.mux.Lock()

	readBufLen := int64(len(h.readBuffer))
	destLen := int64(len(dest))

	if h.readIndex >= readBufLen {
		newFrame := make([]byte, (h.readIndex-readBufLen)+destLen)
		n, err = io.ReadFull(h.source, newFrame)
		h.readBuffer = append(h.readBuffer, newFrame[:n]...)
		if h.readIndex < int64(len(h.readBuffer)) {
			copy(dest, h.readBuffer[h.readIndex:])
		} else {
			err = io.EOF
		}
		h.readIndex += int64(n) - (h.readIndex - readBufLen)
	} else {
		var unbufferedLen int

		endIndex := h.readIndex + destLen
		if endIndex > readBufLen {
			endIndex = readBufLen
		}
		bufferedPart := h.readBuffer[h.readIndex:endIndex]

		if destLen-int64(len(bufferedPart)) > 0 {
			unbufferedPart := make([]byte, destLen-int64(len(bufferedPart)))
			unbufferedLen, err = h.source.Read(unbufferedPart)
			unbufferedPart = unbufferedPart[:unbufferedLen]
			copy(dest, append(bufferedPart, unbufferedPart...))
			n = len(bufferedPart) + unbufferedLen
			h.readBuffer = append(h.readBuffer, unbufferedPart...)
		} else {
			copy(dest, bufferedPart)
			n = len(bufferedPart)
		}

		h.readIndex += int64(n)
		if h.readIndex >= h.totalSize {
			err = io.EOF
		}
	}

	if err != nil {
		if err == io.EOF && !h.done {
			h.source.Close()
			h.buffered = true
			h.done = true
		} else if err == http.ErrBodyReadAfterClose {
			err = io.EOF
		}
	}

	h.mux.Unlock()
	return
}

func (h *HttpReadSeeker) Seek(offset int64, whence int) (pos int64, err error) {
	h.mux.Lock()

	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos = h.readIndex + offset
	case io.SeekEnd:
		pos = h.totalSize + offset
	}

	if pos < 0 || pos > h.totalSize {
		pos = h.readIndex
		err = errOutOfSize
	} else {
		if pos == h.totalSize {
			h.done = true
		} else {
			h.done = false
		}
		h.readIndex = pos
	}

	h.mux.Unlock()
	return
}

func (h *HttpReadSeeker) IsDone() bool {
	if h == nil {
		return false
	}
	return h.done
}

func (h *HttpReadSeeker) Progress() float64 {
	if h == nil {
		return 0
	}
	return float64(h.readIndex) / float64(h.totalSize)
}

func (h *HttpReadSeeker) BufferingProgress() float64 {
	if h == nil {
		return 0
	}
	return float64(len(h.readBuffer)) / float64(h.totalSize)
}
