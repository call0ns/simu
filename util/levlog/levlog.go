package levlog

import (
	"fmt"
	//	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

const (
	LevelFatal = 1 // Highest level: important stuff down
	LevelError = 2 // For example application crashes / exceptions.
	LevelWarn  = 3 // Incorrect behavior but the application can continue
	LevelInfo  = 4 // Normal behavior like mail sent, user updated profile etc.
	LevelDebug = 5 // Executed queries, user authenticated, session expired
	LevelTrace = 6 // Begin method X, end method X etc
)

type LevelLogger struct {
	Trace *log.Logger
	Debug *log.Logger
	INFO  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Fatal *log.Logger
}

var logger LevelLogger = LevelLogger{
	Trace: log.New(ioutil.Discard, "[TRACE] ", log.Ldate|log.Ltime|log.Lshortfile),
	Debug: log.New(ioutil.Discard, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
	INFO:  log.New(os.Stdout, "[INFO ] ", log.Ldate|log.Ltime|log.Lshortfile),
	Warn:  log.New(os.Stdout, "[WARN ] ", log.Ldate|log.Ltime|log.Lshortfile),
	Error: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
	Fatal: log.New(os.Stderr, "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile),
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// Set log level. Can change during runtime
func Start(level int) int {
	if level > LevelTrace {
		level = LevelTrace
	}
	if level < LevelFatal {
		level = LevelFatal
	}

	traceHandle := ioutil.Discard
	debugHandle := ioutil.Discard
	infoHandle := ioutil.Discard
	warnHandle := ioutil.Discard
	errorHandle := os.Stderr
	fatalHandle := os.Stderr

	switch level {
	case LevelTrace:
		traceHandle = os.Stdout
		debugHandle = os.Stdout
		infoHandle = os.Stdout
		warnHandle = os.Stdout
	case LevelDebug:
		debugHandle = os.Stdout
		infoHandle = os.Stdout
		warnHandle = os.Stdout
	case LevelInfo:
		infoHandle = os.Stdout
		warnHandle = os.Stdout
	case LevelWarn:
		warnHandle = os.Stdout
	default: // always on
		//	case LevelError:
		//	case LevelFatal:

	}
	logger.Trace = log.New(traceHandle, "[TRACE] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Debug = log.New(debugHandle, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.INFO = log.New(infoHandle, "[INFO ] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Warn = log.New(warnHandle, "[WARN ] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Error = log.New(errorHandle, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Fatal = log.New(fatalHandle, "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile)

	return level
}

//Most detail in printing.
func Trace(a ...interface{}) {
	logger.Trace.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Tracef(format string, a ...interface{}) {
	logger.Trace.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Tracecd(depth int, a ...interface{}) {
	logger.Trace.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Tracefcd(depth int, format string, a ...interface{}) {
	logger.Trace.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

//For troubleshooting but not too much printout.
func Debug(a ...interface{}) {
	logger.Debug.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Debugf(format string, a ...interface{}) {
	logger.Debug.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Debugcd(depth int, a ...interface{}) {
	logger.Debug.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Debugfcd(depth int, format string, a ...interface{}) {
	logger.Debug.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Info(a ...interface{}) {
	logger.INFO.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Infof(format string, a ...interface{}) {
	logger.INFO.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Infocd(depth int, a ...interface{}) {
	logger.INFO.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Infofcd(depth int, format string, a ...interface{}) {
	logger.INFO.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Warning(a ...interface{}) {
	logger.Warn.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Warningf(format string, a ...interface{}) {
	logger.Warn.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Warningcd(depth int, a ...interface{}) {
	logger.Warn.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Warningfcd(depth int, format string, a ...interface{}) {
	logger.Warn.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Error(a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Errorf(format string, a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

func Errorcd(depth int, a ...interface{}) {
	logger.Error.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
}

func Errorfcd(depth int, format string, a ...interface{}) {
	logger.Error.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
}

//Severe problem.
func Fatal(a ...interface{}) {
	logger.Fatal.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
	os.Exit(1)
}

func Fatalf(format string, a ...interface{}) {
	logger.Fatal.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
	os.Exit(1)
}

func Fatalcd(depth int, a ...interface{}) {
	logger.Fatal.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprint(a...)))
	os.Exit(1)
}

func Fatalfcd(depth int, format string, a ...interface{}) {
	logger.Fatal.Output(2+depth, fmt.Sprintf("%s\n", fmt.Sprintf(format, a...)))
	os.Exit(1)
}

//Function start. Package and function will be output.
func Started() {
	logger.Trace.Output(2, fmt.Sprintf("[%s] Started.\n", getCaller()))
}

//Function return. Package and function will be output.
func Completed() {
	logger.Trace.Output(2, fmt.Sprintf("[%s] Completed.\n", getCaller()))
}

//Program exit with error. Package and function will be output.
func Exit(a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("[%s] Exit error: %s\n", getCaller(), fmt.Sprint(a...)))
}

func Exitf(format string, a ...interface{}) {
	logger.Error.Output(2, fmt.Sprintf("[%s] Exit error: %s\n", getCaller(), fmt.Sprintf(format, a...)))
}

//Print file and line, package and function. Leave a foot print.
func Mark(a ...interface{}) {
	logger.Trace.Output(2, fmt.Sprintf("[%s] %s\n", getCaller(), fmt.Sprint(a...)))
}

func Markf(format string, a ...interface{}) {
	logger.Trace.Output(2, fmt.Sprintf("[%s] %s\n", getCaller(), fmt.Sprintf(format, a...)))
}

func getCaller() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(3, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()

}

//To keep backward compatible with existing log printout.
func Print(v ...interface{}) {
	logger.INFO.Output(2, fmt.Sprintf("%s", fmt.Sprint(v...)))
}

func Println(v ...interface{}) {
	logger.INFO.Output(2, fmt.Sprintf("%s\n", fmt.Sprint(v...)))
}

func Printf(format string, v ...interface{}) {
	logger.INFO.Output(2, fmt.Sprintf("%s\n", fmt.Sprintf(format, v...)))
}
