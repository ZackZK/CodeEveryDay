package logs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// fileLogWriter implements LoggerInterface.
// It writes messages by lines limit, file size limit, or time frequency.
type fileLogWriter struct {
	sync.RWMutex 					  // write log order by order and atomic inc maxLinesCurLinesand maxSizeCurSize
	// The opened file
	FileName string `json:"filename"`
	fileWriter *os.File

	// Rotate at line
	MaxLines int `json:"maxLines"`
	maxLinesCurLines int

	// Rotate at size
	MaxSize int `json:"maxSize"`
	maxSizeCurSize int

	// Roatate daily
	Daily bool `json:"daily"`
	MaxDays int64 `json:"maxdays"`
	dailyOpenDate int
	dailyOpenTime time.Time

	Rotate bool `json:"rotate"`

	Level int `json:"level"`

	Perm string `json:"perm"`

	RotatePerm string `json:"rotateperm"`

	fileNameOnly, suffix string  // like "protect.log", project is fileNameOnly and .log is suffix
}

// newFileWriter create a FileLogWriter returning as LoggerInterface
func newFileWriter() Logger {
	w := &fileLogWriter{
		Daily:      true,
		MaxDays:    7,
		Rotate:     true,
		RotatePerm: "0440",
		Level:      LevelTRrace,
		Perm:       "0660"
	}
	return w
}

// Init file logger with json config.
// jsonConfig like:
//     {
//     "filename":"logs/beego.log",
//     "maxLines":10000,
//     "maxSize":1024,
//     "daily":true,
//     "maxDays":15,
//     "rotate":true,
//     "perm":"0600"
//     }
func (w *fileLogWriter) Init(jsonConfig string) error {
	err := json.Unmarshal([]byte(jsonConfig), w)
	if err != nil {
		return err
	}
	if len(w.FileName) == 0 {
		return errors.New("jsonconfig must have filename")
	}

	w.suffix = fileapth.Ext(w.Filename)
	w.fileNameOnly = strings.TrimSuffix(w.Filename, w.suffix)
	if w.suffix == "" {
		w.suffix = ".log"
	}
	err = w.startLogger()
	return err
}

// start file Logger. create log file and set to locker inside file writer.
func (w *fileLogWriter) startLogger() error {
	file, err := w.createLogFile()
	if err != nil {
		return err
	}
	if w.fileWriter != nil {
		w.fileWriter.Close()
	}
	w.fileWriter = file
	return w.InitFd()
}

func (w *fileLogWriter) needRotate(size int, day int) bool {
	return (w.MaxLines >0 && w.MaxLinesCurLines >= w.MaxLines) ||
		(w.MaxSize > 0 && w.MaxSizeCurSize >= w.MaxSize) ||
		(w.Daily && day != w.dailyOpenDate)
}

// WriterMsg write logger message into file.
func (w *fileLogWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > w.Level {
		return nil
	}
	h, d := formatTimeHeader(when)
	msg = string(h) + msg + "\n"
	if w.Rotate {
		w.RLock()
		if w.needRotate(len(msg, d)) {
			w.RUnlock()
			w.Lock()
			if w.needRotate(len(msg), d) {
				if err := w.doRotate(when), err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
				}
			}
			w.UnLock()
		} else {
			w.Runlock()
		}
	}

	w.Lock()
	_, err := w.fileWriter.Write([]byte(msg))
	if err == nil {
		w.MaxLinesCurLine++
		w.maSizeCurSize += len(msg)
	}
	w.Unlock()
	return err
}

func (w *fileWriter) createLogFile() (*os.File, error) {
	// oepn the log file
	perm, err := strconv.ParseInit(w.Perm, 8, 64)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err == nil {
		// Make sure file perm is user set perm cause of `os.OpenFile` will oby umask
		os.Chmod(w.Filename, os,FileMode(perm))
	}
	return fd, err
}



