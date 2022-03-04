package logger

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/param"
	"github.com/gookit/goutil/strutil"
)

var logBasPath string

// LogResult struct
type LogResult struct {
	FromLineNum int32  `json:"fromLineNum"`
	ToLineNum   int32  `json:"toLineNum"`
	LogContent  string `json:"logContent"`
	IsEnd       bool   `json:"isEnd"`
}

// JavaClassName get
func (LogResult) JavaClassName() string {
	return "com.xxl.job.core.biz.model.LogResult"
}

// LogBasePath dir
func LogBasePath() string {
	return logBasPath
}

// GlueSourcePath file path
func GlueSourcePath() string {
	return logBasPath + "/" + constants.GlueSourceName
}

// SetLogBasePath dir
func SetLogBasePath(path string) {
	logBasPath = path
}

// GetLogPath dir path.
func GetLogPath(nowTime time.Time) string {
	return logBasPath + "/" + nowTime.Format(constants.DateFormat)
}

// OpenLogFile handler
func OpenLogFile(logfile string) (*os.File, error) {
	return os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
}

// LogfilePath get log file path.
func LogfilePath(logId int64) string {
	nowTime := time.Now()

	return logBasPath + "/" + nowTime.Format(constants.DateFormat) + "_" + LogfileName(logId)
}

// LogfileName build
func LogfileName(logId int64) string {
	return fmt.Sprintf("%d", logId) + ".log"
}

// InitLogPath dir
func InitLogPath() error {
	Infof("job logs base path dir: %s", logBasPath)

	_, err := os.Stat(LogBasePath())
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logBasPath, os.ModePerm)
	}

	return err
}

// LogJobf info to file
func LogJobf(ctx context.Context, tpl string, args ...interface{}) {
	LogJob(ctx, fmt.Sprintf(tpl, args...))
}

// LogJob info to file
func LogJob(ctx context.Context, args ...interface{}) {
	val := ctx.Value(constants.CtxParamKey)
	if val == nil {
		Errorf("the job task ctx %s not exists", constants.CtxParamKey)
		return
	}

	if cjp, ok := val.(*param.CtxJobParam); ok {
		nowTime := time.Now()

		// TODO use buffer pool
		var buffer bytes.Buffer
		buffer.WriteString(nowTime.Format(constants.DateTimeFormat))
		buffer.WriteString(" [")
		buffer.WriteString(cjp.JobName)
		buffer.WriteString("#")
		buffer.WriteString(cjp.JobFunc)
		buffer.WriteString("]-[job:")

		buffer.WriteString(strutil.MustString(cjp.JobID))
		buffer.WriteString(", task:")
		buffer.WriteString(strutil.MustString(cjp.JobID))
		buffer.WriteString("] ")

		if len(args) > 0 {
			buffer.WriteString(fmt.Sprintln(args...))
		} else {
			buffer.WriteString("\n")
		}

		pathPrefix := GetLogPath(nowTime)
		// up: 不创建子目录
		writeLogV2(pathPrefix+"_"+LogfileName(cjp.LogID), buffer.String())
		// writeLog(pathPrefix, LogfileName(cjp.LogID), buffer.String())
	}
}

// OpenDirFile open file in the dir
// func OpenDirFile(logDir, fileName string) (*os.File, error) {
// 	logfilePath := logDir + "/" + fileName
//
// 	// ensure log dir is created
// 	_, err := os.Stat(logDir)
// 	if err != nil && os.IsNotExist(err) {
// 		err = os.MkdirAll(logBasPath, os.ModePerm)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	return OpenLogFile(logfilePath)
// }

// func writeLog(logPath, logFile, msg string) error {
// 	if !strutil.IsBlank(logFile) {
// 		file, err := OpenDirFile(logPath, logFile)
// 		if err != nil {
// 			return err
// 		}
//
// 		if file != nil {
// 			defer file.Close()
// 			res, err := file.Write([]byte(msg))
// 			if err != nil {
// 				return err
// 			}
// 			if res <= 0 {
// 				return errors.New("write log failed")
// 			}
// 		}
// 	}
// 	return nil
// }

func writeLogV2(logFile, msg string) error {
	if !strutil.IsBlank(logFile) {
		file, err := OpenLogFile(logFile)
		if err != nil {
			return err
		}

		if file != nil {
			defer file.Close()
			res, err := file.Write([]byte(msg))
			if err != nil {
				return err
			}
			if res <= 0 {
				return errors.New("write log failed")
			}
		}
	}
	return nil
}

// ReadLog from log file
func ReadLog(logDateTim, logId int64, fromLineNum int32) (line int32, content string) {
	nowTime := time.Unix(logDateTim/1000, 0)
	pathPrefix := GetLogPath(nowTime)

	// fileName := GetLogPath(nowTime) + "/" + fmt.Sprintf("%d", logId) + ".log"
	fileName := pathPrefix + "_" + fmt.Sprintf("%d", logId) + ".log"
	file, err := os.Open(fileName)
	totalLines := int32(1)

	var buffer bytes.Buffer
	if err == nil {
		defer file.Close()

		rd := bufio.NewReader(file)
		for {
			line, err := rd.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}

			if totalLines >= fromLineNum {
				buffer.WriteString(line)
			}
			totalLines++
		}
	}
	return totalLines, buffer.String()
}
