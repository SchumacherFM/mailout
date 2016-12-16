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

const stdOut = "stdout"
const stdErr = "stderr"

// MultiMessageSeparator used in WriteTo function in the message slice type to
// separate between multiple messages in a log file.
var MultiMessageSeparator = []byte("\n\n================================================================================\n\n")

// Logger logs emails and errors. If nil, nothing gets logged.
type Logger struct {
	hosts []string

	// MailDir writes mails into this directory. If set to "stderr" or "stdout"
	// then the output will be forwarded to those ports.
	MailDir string

	// ErrDir writes error log file into this directory. If set to "stderr" or
	// "stdout" then the output will be forwarded to those ports.
	ErrDir string
	errlog *log.Logger
	// ErrFile full file path to the error log file
	ErrFile string
	// errfile used for syncing to disk after logging a message
	errFile io.Writer
}

// New creates a new logger by a given directory. If the directory does not
// exists it will be created recursively. Empty directory means a valid nil
// logger.
func New(mailDir, errDir string) Logger {
	if mailDir == "" && errDir == "" {
		return Logger{}
	}
	return Logger{
		MailDir: mailDir,
		ErrDir:  errDir,
	}
}

// IsNil returns true if the Logger is empty which means no paths are set.
func (l Logger) IsNil() bool {
	return l.MailDir == "" && l.ErrDir == ""
}

// Init creates directories and the error log file
func (l Logger) Init(hosts ...string) (Logger, error) {
	// clean host name
	rpl := strings.NewReplacer("/", "", string(os.PathSeparator), "", ":", "", "https", "", "http", "")
	for i, h := range hosts {
		hosts[i] = rpl.Replace(h)
	}
	l.hosts = hosts

	{
		var mailDir = l.MailDir
		var errDir = l.ErrDir
		if mailDir == stdOut || mailDir == stdErr {
			mailDir = ""
		}
		if errDir == stdOut || errDir == stdErr {
			errDir = ""
		}

		for _, dir := range [2]string{mailDir, errDir} {
			if dir == "" {
				continue
			}

			if !isDir(dir) {
				if err := os.MkdirAll(dir, 0700); err != nil {
					return Logger{}, fmt.Errorf("Cannot create directory %q because of: %s", dir, err)
				}
			}
		}
	}

	switch {
	case l.IsNil():
		return Logger{}, nil
	case l.ErrDir == stdErr:
		l.errFile = os.Stderr
	case l.ErrDir == stdOut:
		l.errFile = os.Stdout
	case l.ErrDir == "":
		return l, nil
	}

	var err error
	if l.errFile == nil { // might contain os.Std*
		l.ErrFile = path.Join(l.ErrDir, fmt.Sprintf("mail_errors_%s.log", strings.Join(hosts, "_")))
		l.errFile, err = os.OpenFile(l.ErrFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if l.errFile == nil {
			l.errFile = os.Stderr
		}
	}

	l.errlog = log.New(l.errFile, "", log.LstdFlags)
	return l, err
}

// NewWriter creates a new file with a file name consisting of a time
// stamp. If it fails to create a file it returns a nilWriteCloser
// and does not log anymore any data. Guaranteed to not return nil.
func (l Logger) NewWriter() io.WriteCloser {

	switch {
	case l.IsNil():
		return nilWriteCloser{}
	case l.MailDir == stdErr:
		return os.Stderr
	case l.MailDir == stdOut:
		return os.Stdout
	case l.MailDir == "":
		return nilWriteCloser{}
	}

	fName := fmt.Sprintf("%s%smail_%s_%d.txt", l.MailDir, string(os.PathSeparator), strings.Join(l.hosts, "_"), time.Now().UnixNano())
	f, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		l.Errorf("failed to create %q with error: %s", fName, err)
		return nilWriteCloser{}
	}
	return f
}

// Errorf writes into the error log file. If the logger is nil
// no writes will happen.
func (l Logger) Errorf(format string, v ...interface{}) {

	if l.errlog == nil || l.ErrDir == "" {
		return
	}
	l.errlog.Printf(format, v...)

	// do not sync on os.Std*
	if f, ok := l.errFile.(*os.File); ok && f != nil && f != os.Stderr && f != os.Stdout {
		if err := f.Sync(); err != nil && err != os.ErrInvalid {
			// so what now?
			println("ErrFile", l.ErrFile, " sync to disk error:", err.Error())
		}
	}
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return fileInfo != nil && fileInfo.IsDir() && err == nil
}

// nilWriteCloser used as a backup to not return a nil interface in function
// NewWriter()
type nilWriteCloser struct {
	io.WriteCloser
}

func (wc nilWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (wc nilWriteCloser) Close() error {
	return nil
}
