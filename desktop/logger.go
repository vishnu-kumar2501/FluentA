package desktop

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const defaultLogMaxBytes = 512 * 1024

type rotatingFileWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	file     *os.File
	size     int64
}

func newRotatingFileWriter(path string, maxBytes int64) (*rotatingFileWriter, error) {
	if maxBytes <= 0 {
		maxBytes = defaultLogMaxBytes
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	var size int64
	if err == nil {
		size = info.Size()
	}
	return &rotatingFileWriter{
		path:     path,
		maxBytes: maxBytes,
		file:     f,
		size:     size,
	}, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.size+int64(len(p)) > w.maxBytes && w.size > 0 {
		_ = w.file.Close()
		w.file = nil
		_ = os.Rename(w.path, w.path+".1")
		f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return 0, err
		}
		w.file = f
		w.size = 0
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

type bestEffortMultiWriter struct {
	primary   io.Writer
	secondary io.Writer
}

func (m *bestEffortMultiWriter) Write(p []byte) (int, error) {
	if m.secondary != nil {
		_, _ = m.secondary.Write(p)
	}
	return m.primary.Write(p)
}

// setupFileLogger attaches a rotating log writer under <dataDir>/logs/app.log
// alongside stderr (when available), returning a cleanup func that restores default logging.
func setupFileLogger(dataDir string) (func(), error) {
	if dataDir == "" {
		dataDir = AppDataDir()
	}
	logPath := filepath.Join(dataDir, "logs", "app.log")
	rw, err := newRotatingFileWriter(logPath, defaultLogMaxBytes)
	if err != nil {
		return func() {}, err
	}
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(&bestEffortMultiWriter{primary: rw, secondary: prevOut})
	return func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		_ = rw.Close()
	}, nil
}
