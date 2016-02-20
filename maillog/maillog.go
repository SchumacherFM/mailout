package maillog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

// Logger logs emails and errors. If nil, nothing gets logged.
type Logger struct {
	path    string
	errlog  *log.Logger
	ErrFile string
	hosts   []string
}

// New creates a new logger by a given directory. If the directory does not exists
// it will be created recursively. Empty directory means a valid nil logger.
func New(directory string) *Logger {
	if directory == "" {
		return nil
	}
	return &Logger{
		path: directory,
	}
}

// Init creates directories and the error log file
func (l *Logger) Init(hosts ...string) (*Logger, error) {
	if l == nil {
		return nil, nil
	}
	l.hosts = hosts
	if false == isDir(l.path) {
		if err := os.MkdirAll(l.path, 0700); err != nil {
			return nil, fmt.Errorf("Cannot create directory %q because of: %s", l.path, err)
		}
	}

	l.ErrFile = path.Join(l.path, fmt.Sprintf("errors_%s_%d.log", strings.Join(hosts, "_"), time.Now().Unix()))

	f, err := os.OpenFile(l.ErrFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}

	l.errlog = log.New(f, "", log.LstdFlags)

	return l, nil
}

// NewWriter creates a new file with a file name consisting of a time stamp.
// If it fails to create a file it returns a nilWriteCloser and does not log
// anymore any data.
func (l *Logger) NewWriter() io.WriteCloser {
	if l == nil {
		return nilWriteCloser{}
	}
	fName := fmt.Sprintf("%s%smail_%s_%d.txt", l.path, string(os.PathSeparator), strings.Join(l.hosts, "_"), time.Now().UnixNano())
	f, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		l.Errorf("failed to create %q with error: %s", fName, err)
		return nilWriteCloser{}
	}
	return f
}

// Errorf writes into the error log file. If the logger is nil
// no write will happen.
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l == nil {
		return
	}
	l.errlog.Printf(format, v...)
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return fileInfo != nil && fileInfo.IsDir() && err == nil
}

type nilWriteCloser struct {
	io.WriteCloser
}

func (wc nilWriteCloser) Write(p []byte) (n int, err error) {
	return
}

func (wc nilWriteCloser) Close() error {
	return nil
}
