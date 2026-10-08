package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Rotation policy, fixed by research R17: 10 MB per file, three kept.
const (
	MaxBytes    = 10 << 20
	KeptBackups = 3
)

// Kind names one of the four sinks the bottom panel surfaces.
type Kind string

const (
	Application  Kind = "application"  // Console tab
	Modification Kind = "modification" // Modification Logs tab — written as LDIF
	Search       Kind = "search"       // Search Logs tab
	Errors       Kind = "errors"       // Errors tab
)

// AllKinds is the fixed set of sinks. There is no dynamic sink registration:
// a log with no tab is a log nobody reads.
var AllKinds = []Kind{Application, Modification, Search, Errors}

// Sink is an append-only, size-rotated file that redacts before it writes.
//
// The redactor lives inside the sink rather than at each call site so that
// "redaction happens at write time" is a property of the type, not a habit of
// its callers: there is no way to reach the file descriptor without passing
// through Redact (FR-093, contract X7).
type Sink struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	size     int64
	maxBytes int64
	keep     int
	redactor *Redactor
}

// OpenSink opens (creating if needed) the file at path.
func OpenSink(path string, redactor *Redactor) (*Sink, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat log %s: %w", path, err)
	}
	return &Sink{
		path:     path,
		file:     f,
		size:     info.Size(),
		maxBytes: MaxBytes,
		keep:     KeptBackups,
		redactor: redactor,
	}, nil
}

// Write redacts p, then appends it, rotating first if the file is full.
// It satisfies io.Writer so a slog handler can be pointed straight at a sink.
func (s *Sink) Write(p []byte) (int, error) {
	redacted := s.redactor.Redact(p)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.size+int64(len(redacted)) > s.maxBytes {
		if err := s.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := s.file.Write(redacted)
	s.size += int64(n)
	// The caller wrote len(p) bytes as far as it is concerned; reporting the
	// redacted length would look like a short write to io.Writer users.
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Record writes one timestamped line. Callers that already produce LDIF (the
// modification log) use Write directly so their bytes stay valid LDIF.
func (s *Sink) Record(format string, args ...any) error {
	line := fmt.Sprintf("%s %s\n", time.Now().UTC().Format(time.RFC3339Nano), fmt.Sprintf(format, args...))
	_, err := s.Write([]byte(line))
	return err
}

// rotate renames the current file to .1, shifting existing backups down and
// discarding anything past keep. The caller holds the lock.
func (s *Sink) rotate() error {
	if err := s.file.Close(); err != nil {
		return err
	}
	oldest := fmt.Sprintf("%s.%d", s.path, s.keep)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := s.keep - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", s.path, i)
		to := fmt.Sprintf("%s.%d", s.path, i+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(s.path, s.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	s.file, s.size = f, 0
	return nil
}

// Path is the sink's current file, for the "reveal in file manager" affordance.
func (s *Sink) Path() string { return s.path }

// Close flushes and closes the underlying file.
func (s *Sink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}

// Logs is the set of open sinks, shared by every package that records anything.
type Logs struct {
	Redactor *Redactor
	sinks    map[Kind]*Sink
}

// Paths locates the application's on-disk directories.
type Paths struct {
	Data string // profiles, trust store, history, schema projects
	Logs string // the four sinks
}

// DefaultPaths resolves the per-user application directories, honouring
// XDG_DATA_HOME / XDG_STATE_HOME on Linux and the OS conventions elsewhere.
func DefaultPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	data := filepath.Join(base, "open-ldap-studio")

	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		state = data
	} else {
		state = filepath.Join(state, "open-ldap-studio")
	}
	return Paths{Data: data, Logs: filepath.Join(state, "logs")}, nil
}

// Open opens all four sinks under paths.Logs.
func Open(paths Paths) (*Logs, error) {
	r := NewRedactor()
	l := &Logs{Redactor: r, sinks: make(map[Kind]*Sink, len(AllKinds))}
	for _, k := range AllKinds {
		s, err := OpenSink(filepath.Join(paths.Logs, string(k)+".log"), r)
		if err != nil {
			_ = l.Close()
			return nil, err
		}
		l.sinks[k] = s
	}
	return l, nil
}

// Sink returns the named sink.
func (l *Logs) Sink(k Kind) *Sink { return l.sinks[k] }

// Close closes every open sink, reporting the first failure.
func (l *Logs) Close() error {
	var firstErr error
	for _, s := range l.sinks {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
