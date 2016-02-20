package maillog

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"
)

// New creates a new logger by a given directory. If the directory does not exists
// it will be created recursively. Empty directory means a valid nil logger.
func New(directory string) (*Logger, error) {

	if directory == "" {
		return nil, nil
	}

	if false == isDir(directory) {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return nil, fmt.Errorf("Cannot create directory %q because of: %s", directory, err)
		}
	}

	logFile := path.Join(directory, fmt.Sprintf("errors_%s.log", time.Now()))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}

	l := &Logger{
		path:    directory,
		errlog:  log.New(f, "", log.LstdFlags),
		ErrFile: logFile,
	}
	return l, nil
}

// Logger logs emails and errors. If nil, nothing gets logged.
type Logger struct {
	path    string
	errlog  *log.Logger
	ErrFile string
}

// Write creates a new file with a file name consisting of a time stamp.
// Writes the data into that file and closes the file afterwards.
// Each write creates a new file.
func (l *Logger) Write(p []byte) (n int, err error) {
	if l == nil {
		return
	}
	fName := fmt.Sprintf("%s%smail_%s.txt", l.path, string(os.PathSeparator), time.Now())
	f, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		l.Errorf("failed to create %q with error: %s", fName, err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			l.Errorf("failed to close %q with error: %s", fName, err)
		}
	}()

	if n, err = f.Write(p); err != nil {
		l.Errorf("failed to write to %q with error: %s", fName, err)
	}
	return
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
