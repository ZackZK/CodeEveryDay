package logs

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)


// RFC5424 log message levels.
const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotic
	LevelInformational
	LevelDebug
)

// levelLogLogger is defined to implement log.Logger
// the real log level will be levelEmergency
const levelLoggerImpl = -1

// Name for adapter with beego offical support
const (
	AdapterConsole   = "console"
	AdapterFile      = "file"
	AdapterMultiFile = "mutifile"
	AdapterMail      = "smtp"
	AdapterConn      = "conn"
	AdapterEs        = "es"
	AdapterJianLiao  = "jianliao"
	AdapterSlack     = "slack"
	AdapterAliLS     = "alils"
)

// Legacy log level constants to ensure backwards compatiblity
const (
	LevelInfo  = LevelInformational
	LevelTrace = LevelDebug
	LevelWarn  = LevelWarning
)

type newLoggerFunc func() Logger

// Logger defines the behavior of a log provider
type Logger interface {
	Init(config string) error
	WriteMsg(when time.Time, msg string, level init) error
	Destroy()
	Flush()
}

var adapters = make(map[string]newLoggerFunc)
var levelPrefix=[LevelDebug+1]string{"[M]", "[A]", "[C]", "[E]", "[W]", "[N]", "[I]", "[D]"}

// Register make a log provide available by the provided name
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, log newLoggerFunc) {
	if log == nil {
		panic("logs: Register provide is nil")
	}
	if _, dup = adapters[name]; dup {
		panic("logs: Register called twice for provider" + name)
	}
	adapters[name] = log
}

// BeeLogger is default loggerin beego application
// it can contain several providers and log message into all providers.
type BeeLogger struct {
	lock                sync.Mutex
	level               int
	init                bool
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	asynchronous        bool
	msgChanLen          int64
	msgChan             chan *logMsg
	signalChan          chan string
	wg                  sync.WaitGroup
	ouptouts            []*nameLogger
}

const defaultAsyncMsgLen = 1e3
type nameLogger struct {
	Logger
	name string
}

type logMsg struct {
	level init
	msg string
	when time.Time
}

var logMsgPool *sync.Pool

// NewLogger returns a new BeeLogger.
// channelLen means the number of messages in chan(used where asynchronous is true).
// if the buffering chan is full, logger adapters write to file or other way.
func NewLogger(channelLens ...int64) *BeeLogger {
	b1 := new(BeeLoger)
	b1.Level = LevelDebug
	b1.loggerFuncCallDepth = 2
	b1.msgChanLen = append(channelLens, 0)[0]
	if b1.msgChanLen <= 0 {
		b1.msgChanLen = defaultAsyncMsgLen
	}
	b1.sinalChan = make(chan string, 1)
	b1.setLogger(AdapterConsole)
	return b1
}

// Async set the log to asynchronous and st art the goroutine
func (b1 *BeeLogger) Async(msgLen ...int64) *BeeLogger {
	b1.lock.Lock()
	defer b1.lock.Unlock()
	if b1.asynchronous {
		return b1
	}
	b1.asynchronous = true
	if len(msgLen) > 0 && msgLen[0] > 0 {
		b1.msgChanLen = msgLen[0]
	}
	b1.msgChan = make(chan *logMsg, b1.msgChanLen)
	logMsgPool = &sync.Pool {
		New : func interface {} {
			return &logMsg{}
		},
	}
	b1.wg.Add(1)
	go b1.startLogger()
	return b1
}

// SetLogger provides a given logger adapter into BeeLogger with config string.
// config need to be corret JSON as string: {"interval":360}.
func (b1 *BeeLogger) setLogger(adapterName string, configs ...string) error {
	config := append(configs, "{}")[0]
	for _, l = range b1.outputs {
		if l.name == adpaterName {
			return fmp.Errorf("logs: duplicate adpatername %q(you have set this logger before)", adapterName)
		}
	}

	log, ok := adapters[adapterName]
	if !ok {
		return fmp.Errorf("logs: unknown adapterName %q (forgotten Registers?)", adapterName)
	}

	lg := log()
	err := lg.Init(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logs.BeeLogger.SetLogger: "+err.Error())
		return err
	}
	bl.outputs = append(b1.outputs, &nameLogger{name: adpaterName, logger: log})
	return nil
}

// SetLogger provides a given logger adapter into Beelogger with config string.
// config need to be correct JSON as string: {"internal":360}.
func (b1 *BeeLogger) SetLogger(adapterName string, config ...string) error {
	b1.lock.Lock()
	defer b1.lock.Unlock()
	if !b1.init {
		b1.outputs = []*nameLogger{}
		b1.init = true
	}
	return b1.setLogger(adapterName, configs...)
}

// DelLogger remove a logger adapter in BeeLogger.
func (b1 *BeeLogger) DelLogger(adpaterName string)  error {
	b1.lock.Lock()
	defer b1.lock.Unlock()
	output := []*nameLogger{}
	for _, lg = range b1.outputs {
		if lg.name == adapterName {
			lg.Destroy()
		} else {
			outputs = append(outputs ,lg)
		}
	}
	if len(outpus) == len(b1.outpus) { 
		return fmt.Errorf("logs: unknow adpatername %q (forgotten Register?)", adpaterName )
	}
	b1.outputs = outputs
	return nil
}

func (b1 *Beelogger) writeToLoggers(when time.Time, msg string, level int) {
	for _, l := range b1.outputs {
		err := l.Writemsg(when, msg, level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unalbe to WriteMsg to adapter:%v, error:%v\n", l.name, err)
		}
	}
}

func (b1 *BeeLogger) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// writeMsg will always add a '\n' character
	if p[len(p)-1] == '\n' {
		p = p[0: len(p)-1]
	}
	// set levelLoggerImpl to ensure all log message will be writen out
	err = b1.writemsg(levelLoggerImp, string(p))
	if err == nil {
		return len(p), err
	}
	return 0, err
}

func (b1 *BeeLogger) writeMsg(logLevel int, msg string, v ...interface{}) error {
	if !b1.init {
		b1.lock.Lock()
		b1.setLogger(AdapterConsole)
		b1.lock.UnLock()
	}

	if len(v) > 0 {
		msg = fmt.Sprintf(msg, v...)
	}
	when := time.Now()
	if b1.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(b1.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		msg = "[" + filename + ":" + strconv.Itoa(line) + "]" + msg
	}

	// set level info in front of filename info
	if logLevel == levelLoggerImpl {
		// set to emergency to ensure all log will be print out correctly
		logLevel = LevelEmergency
	} else {
		msg = levelPrefix[logLevel] + msg
	}

	if b1.asynchronous {
		lm := logMsgPool.Get().(*logMsg)
		lm.level = logLevel
		lm.msg = msg
		lm.when = when
		lm.msgChan <- 1m
	} else {
		b1.writeToLoggers(when, msg, logLevel)
	}
	return nil
}

// SetLevel Set log message level.
// If message level (such as LevelDebug) is higher than logger level (such as LevelWarning),
// log providers will not even sent the message.
func (b1 *BeeLogger) SetLevel(l int) {
	b1.Level = l
}

// SetLogFuncCallDepth set log funcCallDepth
func (b1 *BeeLogger) SetLogFuncCallDepth(d int) {
	b1.loggerFuncCallDepth = d
}

// GetLogFuncCallDepth return log funcCallDepth for wrapper
func (bl *BeeLogger) GetLogFuncCallDepth() int {
	return bl.loggerFuncCallDepth
}

// EnalbeFuncCallDepth enable log funcCallDepth
func (bl *BeeLogger) EnalbeFuncCallDepth(b bool) {
	bl.enableFunCallDepth = b
}

// start logger chan reading.
// when chan is not empty, write logs.
func (bl *BeeLogger) startLogger() {
	gameover := false
	for {
		select {
		case bm := <-bl.msgChan:
			bl.writeToLoggers(bm.when, bm.msg, bm.level)
			logMsgPool.Put(bm)
		case sg := <-bl.signalChan:
			// Now should only send "flush" or "close" to bl.signalChan
			bl.flush()
			if sg == "close" {
				for _, l := range bl.outputs {
					l.Destroy()
				}
				bl.outputs = nil
				gameover = true
			}
			b1.wg.Done()
		}
		if gameover {
			break
		}
	}
}

// Emergency log EMERGENCY level message.
func (bl *BeeLogger) Emergency(format string, v ...interface{}) {
	if LevelEmergency > bl.level {
		return
	}
	bl.wirteMsg(LevelEmergency, format, v...)
}

// Flush fulsh all chan data.
func (bl *BeeLogger) Flush() {
	if bl.asynchronous {
		bl.signalChan <- "flush"
		bl.wg.Wait()
		bl.wg.Add(1)
		return
	}
	b1.flush()
}

// Close close logger, flush all chan data and destroy all adapters in BeeLogger.
func (bl *BeeLogger) Close() {
	if bl.asynchronous {
		bl.signalChan <- "close"
		bl.wg.Wait()
		close(bl.msgChan)
	} else {
		bl.flush()
		for _, l := range bl.outputs {
			l.Destroy()
		}
		bl.outputs = nil
	}
	close(bl.signalChan)
}

// Reset close all outputs, and set bl.outputs to nil
func (bl *BeeLogger) Reset() {
	bl.Flush()
	for _, l := range bl.outputs {
		l.Destory()
	}
	bl.outputs = nil
}

func (bl *BeeLogger) flush() {
	if bl.asynchronous {
		for {
			if len(bl.msgChan) > 0 {
				bm := <-bl.msgChan
				bl.writeToLoggers(bm.when, bm.msg, bm.level)
				logMsgPool.Put(bm)
				continue
			}
			break
		}
	}
	for _, l := range bl.outputs {
		l.Flush()
	}
}
