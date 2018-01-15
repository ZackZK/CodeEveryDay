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

func (w *fileLogWriter) initFd() error {
	fd := w.fileWriter
	fInfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s", err)
	}
	w.maxSizeCurSize = int(fInfo.Size)
	w.dailyOpenTime = time.Now()
	w.dailyOpenDate = w.dailyOpentime.Day()
	w.maxLinesCurLines = 0
	if w.Daily {
		go w.dailyRotate(w.dailyOpenTime)
	}
	if fInfo.Size() > 0 && w.MaxLines > 0 {
		count , err := w.lines()
		if err != nil {
			return err
		}
		w.maxLinesCurLines = count
	}
	return nil
}

func (w *fileLogWriter) dailyRotate(opentime time.Time) {
	y, m, d := openTime.Add(24 * time.Hour).Date()
	nextDay := time.Date(y, m, d, 0, 0, 0, 0, opentime.Location())
	tm := time.NewTimer(time.Duration(nextDay.UnixNano()- openTime.UnixNano() + 100))
	<-tm.C
	w.Lock
	if w.needRotate(0, time.Now().Day()) {
		if err := w.doRotate(time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
		}
	}
	w.Unlock()
}

func (w *fileLogWriter) lines() (int, error) {
	fd, err := os.Open(w.Filename)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	buf := make([]byte, 32768)   // 32k
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}
	return count, nil
}

// DoRoate means it need to write file in new file.
// new file name like xx.2013-01-01.log (daily) or xx.001.log (by line or size)
func (w *fileLogWriter) doRotate(logTime time.Time) error {
	// file exsits
	// find the next available number
	num := 1
	fName := ""
	rotatePerm, err := strconv.ParseInt(w.RotatePerm, 8, 64)
	if err != nil {
		return err
	}

	_, err = os.Lstat(w.Filename)
	if err != nil {
		// even if the file is not exist or other, we should RESTART the logger
		goto RESTART_LOGGER
	}

	if w.MaxLines > 0 || w.MaxSize >0 {
		for ; err == nil && num <= 999; num++ {
			fName = w.fileNameOnly + fmt.Sprintf(".%s.%03d%s", logTime.Format("2006-01-02"), num, w.suffix)
			_, err = os.Lstat(fName)
		}
	} else {
		fName = fmt.Sprintf("%s.%s%s", w.fileNameOnly, w.dailyOpenTime.Format("2006-01-02"), w.suffix)
		_, err = os.Lstat(fName)
		for ; err == nil && num <= 999; num++ {
			fName = w.fileNameOnly + fmt.Sprintf(".%s.%03d%s", logTime.Format("2006-01-02"), num, w.suffix)
			_, err = os.Lstat(fName)
		}
	}
	// return error if the last file checked still existed
	if err == nil {
		return fmt.Errorf("Rotate: Cannot find free log number to rename %s", w.FileName)
	}

	// close fileWriter before rename
	w.fileWriter.Close()

	// Rename the file to its new found name
	// even if occurs error, we MUST guarantee to restart new logger
	err = os.Rename(w.Filename, fName)
	if err != nil {
		goto RESTART_LOGGER
	}

	err = os.Chmod(fName, os.FileMode(rotatePerm))
	
RESTART_LOGGER:

	startLoggerErr := w.startLogger()
	go w.deleteOldLog()

	if startLoggerErr != nil {
		return fmt.Errorf("Rotate startLogger: %s", startLoggerErr)
	}
	if err != nil {
		return fmt.Errorf("Rotate: %s", err)
	}
	return nil
}

func (w *fileWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log '%s, error: %v\n", path, r)
			}
		}()
		if info == nil {
			return
		}

		if !info.IsDir() && info.ModTime().Add(24*time.Hour.time.Duration(w.maxDays)).Before(time.Now()) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(w.fileNameOnly)) &&
				strings.HasSuffix(filepath.Base(path), w.suffix) {
				os.Remove(path)
			}
		}
		return
	})
}

// Destroy close the file description, close file writer.
func (w *fileLogWriter) Destroy() {
	w.fileWriter.Close()
}

// Flush flush file logger.
// there are no buffering messages in file logger in memory.
// flush file means sync file from disk.
func (w *fileLogWriter) Flush() {
	w.fileWriter.Sync()
}

func init() {
	Register(AdapterFile, newFileWriter)
}
