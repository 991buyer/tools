package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"time"
)

const (
	DAY_SECOND = 86400
)

var _defaultWriter *RotateWriter

type RotateWriter struct {
	dir        string
	filename   string
	fp         *os.File
	wg         sync.WaitGroup
	quit       bool
	msgChan    chan string
	serverName string
}

func init() {
	dir := "./log"
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	serverPath, err := exec.LookPath(os.Args[0])
	if err != nil {
		panic(err)
	}

	serverName := filepath.Base(serverPath)

	_defaultWriter = &RotateWriter{
		dir:        dir,
		quit:       false,
		msgChan:    make(chan string, 10000),
		serverName: serverName,
	}

	err = _defaultWriter.NewLogFile(time.Now().Format("2006-01-02.log"))
	if err != nil {
		panic(err)
	}

	go _defaultWriter.StartWriter()

	log.SetOutput(io.MultiWriter(_defaultWriter, os.Stdout))
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	Info("%s logger started", serverName)
}

func (w *RotateWriter) StartWriter() {
	w.wg.Add(1)

	pm12 := DAY_SECOND - time.Now().Unix()%DAY_SECOND + 61 // 第二天
	newFileTimer := time.After(time.Duration(pm12) * time.Second)
	quitTicker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-newFileTimer:
			fname := time.Now().Format("2006-01-02.log")
			w.NewLogFile(fname)

			pm12 = DAY_SECOND - time.Now().Unix()%DAY_SECOND + 61 // 第二天
			newFileTimer = time.After(time.Duration(pm12) * time.Second)

		case msg := <-w.msgChan:
			w.fp.Write([]byte(msg))

		case <-quitTicker.C:
			//退出
			if w.quit {
				if len(w.msgChan) == 0 {
					w.wg.Done()
					return
				}
			}
		}
	}
}

func (w *RotateWriter) Write(output []byte) (int, error) {
	w.msgChan <- string(output)

	return 0, nil
}

func (w *RotateWriter) NewLogFile(fname string) (err error) {
	if w.filename == fname && w.filename != "" {
		return nil
	}

	file, err := os.OpenFile(path.Join(w.dir, fname), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println("err:", err)
		return err
	}

	// Close existing file if open
	if w.fp != nil {
		err = w.fp.Close()
		if err != nil {
			log.Println("err:", err)
		}
	}

	w.fp = file
	w.filename = fname

	return
}

// 把缓存队列里的日志写入磁盘
func Flush() {
	if _defaultWriter != nil && _defaultWriter.quit == false {
		_defaultWriter.quit = true
		_defaultWriter.wg.Wait()
	}
}

func Trace(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[TRACE] "+format, v...))
}

func Debug(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[DEBUG] "+format, v...))
}

func Info(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[INFO ] "+format, v...))
}

func Warn(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[WARN ] "+format, v...))
}

func Error(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[ERROR] "+format, v...))
}

func Alarm(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[ALARM] "+format, v...))
}

func Fatal(format string, v ...interface{}) {
	log.Output(2, fmt.Sprintf("[FATAL] "+format, v...))

	os.Exit(1)
}
