package bytecount

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var (
	liveSec       atomic.Int64
	liveUpAllow   atomic.Uint64
	liveUpDeny    atomic.Uint64
	liveDownAllow atomic.Uint64
	liveDownDeny  atomic.Uint64
	totalUpAllow   atomic.Uint64
	totalUpDeny    atomic.Uint64
	totalDownAllow atomic.Uint64
	totalDownDeny  atomic.Uint64
	rotateMu      sync.Mutex
	onRotate      func(sec int64, upAllow, upDeny, downAllow, downDeny uint64)
)

// Totals — накопленные байты с момента старта процесса (hot path, без 5m окна).
func Totals() (upAllow, upDeny, downAllow, downDeny uint64) {
	return totalUpAllow.Load(), totalUpDeny.Load(), totalDownAllow.Load(), totalDownDeny.Load()
}

// SetRotateHook вызывается при смене секунды (из metrics при init).
func SetRotateHook(fn func(sec int64, upAllow, upDeny, downAllow, downDeny uint64)) {
	onRotate = fn
}

// Rotate сбрасывает завершённую секунду (1 Гц из snapshot — для точности графика).
func Rotate(newSec int64) {
	byteSecRotate(newSec)
}

// Live — текущая незавершённая секунда.
func Live() (sec int64, upAllow, upDeny, downAllow, downDeny uint64) {
	return liveSec.Load(), liveUpAllow.Load(), liveUpDeny.Load(), liveDownAllow.Load(), liveDownDeny.Load()
}

func observeUp(n int, denied bool) {
	if n <= 0 {
		return
	}
	for {
		sec := time.Now().Unix()
		if liveSec.Load() != sec {
			byteSecRotate(sec)
		}
		if liveSec.Load() == sec {
			if denied {
				liveUpDeny.Add(uint64(n))
				totalUpDeny.Add(uint64(n))
			} else {
				liveUpAllow.Add(uint64(n))
				totalUpAllow.Add(uint64(n))
			}
			return
		}
	}
}

func observeDown(n int, denied bool) {
	if n <= 0 {
		return
	}
	for {
		sec := time.Now().Unix()
		if liveSec.Load() != sec {
			byteSecRotate(sec)
		}
		if liveSec.Load() == sec {
			if denied {
				liveDownDeny.Add(uint64(n))
				totalDownDeny.Add(uint64(n))
			} else {
				liveDownAllow.Add(uint64(n))
				totalDownAllow.Add(uint64(n))
			}
			return
		}
	}
}

// ObserveUp — исх, разрешённый трафик.
func ObserveUp(n int) {
	observeUp(n, false)
}

// ObserveUpDenied — исх, заблокированный (403, forbidden MITM и т.п.).
func ObserveUpDenied(n int) {
	observeUp(n, true)
}

// ObserveDown — вх, разрешённый трафик.
func ObserveDown(n int) {
	observeDown(n, false)
}

// ObserveDownDenied — вх, заблокированный.
func ObserveDownDenied(n int) {
	observeDown(n, true)
}

func byteSecRotate(newSec int64) {
	rotateMu.Lock()
	defer rotateMu.Unlock()
	cur := liveSec.Load()
	if newSec <= cur {
		return
	}
	if cur != 0 {
		upA := liveUpAllow.Swap(0)
		upD := liveUpDeny.Swap(0)
		downA := liveDownAllow.Swap(0)
		downD := liveDownDeny.Swap(0)
		if onRotate != nil {
			onRotate(cur, upA, upD, downA, downD)
		}
	} else {
		liveUpAllow.Store(0)
		liveUpDeny.Store(0)
		liveDownAllow.Store(0)
		liveDownDeny.Store(0)
	}
	liveSec.Store(newSec)
}

type readCloserUp struct {
	rc     io.ReadCloser
	denied bool
}

// WrapBodyUp считает тело запроса (клиент → origin).
func WrapBodyUp(rc io.ReadCloser) io.ReadCloser {
	return WrapBodyUpPolicy(rc, false)
}

// WrapBodyUpPolicy — как WrapBodyUp, с учётом allow/deny.
func WrapBodyUpPolicy(rc io.ReadCloser, denied bool) io.ReadCloser {
	if rc == nil {
		return rc
	}
	return readCloserUp{rc: rc, denied: denied}
}

func (r readCloserUp) Read(p []byte) (int, error) {
	n, err := r.rc.Read(p)
	if n > 0 {
		observeUp(n, r.denied)
	}
	return n, err
}

func (r readCloserUp) Close() error {
	return r.rc.Close()
}

type writerDown struct {
	w      io.Writer
	denied bool
}

// WrapWriterDown считает ответ клиенту (origin → клиент).
func WrapWriterDown(w io.Writer) io.Writer {
	return WrapWriterDownPolicy(w, false)
}

// WrapWriterDownPolicy — как WrapWriterDown, с учётом allow/deny.
func WrapWriterDownPolicy(w io.Writer, denied bool) io.Writer {
	if w == nil {
		return w
	}
	return writerDown{w: w, denied: denied}
}

func (w writerDown) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if n > 0 {
		observeDown(n, w.denied)
	}
	return n, err
}
