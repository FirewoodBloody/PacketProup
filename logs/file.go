package logs

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Filelogger struct {
	//定义日志结构体
	level        int
	logPath      string
	logName      string
	file         *os.File
	warnFile     *os.File
	LogDataChan  chan *LogData
	logSplitType bool
	logSplitSize int64
	logNowSplit  string
}

//New接口方法
func NewFilelogger(config map[string]string) (LogInterface, error) {
	var (
		logSplitType bool
		logSplitSize int64
	)
	logPath, ok := config["logPath"]
	if !ok {
		err := fmt.Errorf("not found logPath")
		return nil, err
	}

	logName, ok := config["logName"]
	if !ok {
		err := fmt.Errorf("not found logName")
		return nil, err
	}

	loglevel, ok := config["loglevel"]
	if !ok {
		err := fmt.Errorf("not found loglevel")
		return nil, err
	}
	logChanSize, ok := config["logChanSize"]
	if !ok {
		logChanSize = "50000"
	}
	ChanSize, err := strconv.Atoi(logChanSize)
	if err != nil {
		ChanSize = 50000
	}

	logSplitDateStr, ok := config["logSplitDate"]
	if !ok {
		logSplitSizeStr, ok := config["logSplitSize"]
		if ok {
			logSplitDateStr = "false"
			logSplitType, err = strconv.ParseBool(logSplitDateStr)
			if err != nil {
				logSplitType = false
			}
			logSplitSize, err = strconv.ParseInt(logSplitSizeStr, 10, 64)
			if err != nil {
				logSplitSize = 104857600
			}
		} else {
			logSplitType = true
		}
	} else {
		logSplitType, err = strconv.ParseBool(logSplitDateStr)
		if err != nil {
			logSplitType = true
		}
	}
	logger := &Filelogger{
		level:        GetLevel(loglevel),
		logName:      logName,
		logPath:      logPath,
		LogDataChan:  make(chan *LogData, ChanSize),
		logSplitType: logSplitType,
		logSplitSize: logSplitSize,
		logNowSplit:  time.Now().Format("2006-01-02"),
	}
	logger.rename()
	go logger.writeLogBackground()
	return logger, nil //接口方法定义
}
func (f *Filelogger) refilename() {
	//日志文件的Name的定义
	now := time.Now().Format("20060102")
	filename := fmt.Sprintf("%s/%s_%s.log", f.logPath, f.logName, now)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("open file %s failed ,err:%v", filename, err))
	}
	f.file = file
}

func (f *Filelogger) rewarnfilename() {
	//写错误日志和fatal日志的文件
	now := time.Now().Format("20060102")
	filename := fmt.Sprintf("%s/%s_%s_error.log", f.logPath, f.logName, now)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		panic(fmt.Sprintf("open file %s failed ,err:%v", filename, err))
	}
	f.warnFile = file
}

//日志文件名定义
func (f *Filelogger) rename() {
	f.refilename()
	f.rewarnfilename()
}

//日志写入的分割方法
func (f *Filelogger) writeSplitType(types string, warnFile bool) {
	if types == "date" {
		now := time.Now().Format("2006-01-02")
		if f.logNowSplit == now {
			return
		} else {
			f.logNowSplit = time.Now().Format("2006-01-02")
			f.Close()
			f.rename()
		}

	} else if types == "size" {
		var filename string
		file := f.file
		if warnFile == true {
			file = f.warnFile
		}
		statInfo, err := file.Stat()
		if err != nil {
			return
		}
		fileSize := statInfo.Size()
		if fileSize <= f.logSplitSize {
			return
		}
		if warnFile == false { //待调整大小切分文件名
			now := time.Now().Format("20060102150405")
			filename = fmt.Sprintf("%s/%s_%s.log", f.logPath, f.logName, now)
			file.Close()
			files, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
			if err != nil {
				fmt.Println(err)
			}
			f.file = files
		} else if warnFile == true {
			now := time.Now().Format("20060102150405")
			filename = fmt.Sprintf("%s/%s_%s_error.log", f.logPath, f.logName, now)
			file.Close()
			files, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
			if err != nil {
				fmt.Println(err)
			}
			f.warnFile = files
		}

	} else {
		now := time.Now().Format("2006-01-02")
		if f.logNowSplit == now {
			return
		} else {
			f.Close()
			f.rename()
		}
	}
}

//后台运行写日志线程
func (f *Filelogger) writeLogBackground() {
	var types string
	for logData := range f.LogDataChan {

		if logData.WarnAndFaral == true {
			if f.logSplitType == true {
				types = "date"
				f.writeSplitType(types, logData.WarnAndFaral)
			} else {
				if f.logSplitSize != 0 {
					types = "size"
					f.writeSplitType(types, logData.WarnAndFaral)
				} else {
					types = "date"
					f.writeSplitType(types, logData.WarnAndFaral)
				}
			}
			fmt.Fprintf(f.warnFile, "%s %s [%s:%d:%s] %s\n", logData.TimeStr, logData.LevelStr, logData.FileName, logData.LineNo, logData.FuncName, logData.Messags)

		} else if logData.WarnAndFaral == false {
			if f.logSplitType == true {
				types = "date"
				f.writeSplitType(types, logData.WarnAndFaral)
			} else {
				if f.logSplitSize != 0 {
					types = "size"
					f.writeSplitType(types, logData.WarnAndFaral)
				} else {
					types = "date"
					f.writeSplitType(types, logData.WarnAndFaral)
				}
			}
			fmt.Fprintf(f.file, "%s %s [%s:%d:%s] %s\n", logData.TimeStr, logData.LevelStr, logData.FileName, logData.LineNo, logData.FuncName, logData.Messags)

		}

		//fmt.Fprintf(file, "%s %s [%s:%d:%s] %s\n", logData.TimeStr, logData.LevelStr, logData.FileName, logData.LineNo, logData.FuncName, logData.Messags)
	}
}

//为struct定义借口方法
func (f *Filelogger) Debug(format string, args ...interface{}) {
	logData := LogWrite(f.file, f.level, LogLevelDebug, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

func (f *Filelogger) Trace(format string, args ...interface{}) {
	logData := LogWrite(f.file, f.level, LogLevelTrace, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

func (f *Filelogger) Info(format string, args ...interface{}) {
	logData := LogWrite(f.file, f.level, LogLevelInfo, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

func (f *Filelogger) Warn(format string, args ...interface{}) {
	logData := LogWrite(f.warnFile, f.level, LogLevelWarn, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

func (f *Filelogger) Error(format string, args ...interface{}) {
	logData := LogWrite(f.warnFile, f.level, LogLevelError, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

func (f *Filelogger) Fatal(format string, args ...interface{}) {
	logData := LogWrite(f.warnFile, f.level, LogLevelFatal, format, args...)
	select {
	case f.LogDataChan <- logData:
	default:
	}
}

//关闭日志写入
func (f *Filelogger) Close() {
	f.file.Close()
	f.warnFile.Close()
}

func (f *Filelogger) SetLevel(level int) {
	if level < LogLevelDebug || level > LogLevelFatal {
		level = LogLevelDebug
	}
	f.level = level
}
