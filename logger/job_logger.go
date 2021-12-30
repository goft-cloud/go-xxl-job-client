package logger

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
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
	// split by YMD dir.
	return logBasPath + "/" + nowTime.Format(constants.DateFormat)
}

// InitLogPath dir
func InitLogPath() error {
	Infof("job logs base path dir: %s", logBasPath)

	_, err := os.Stat(GetLogPath(time.Now()))
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logBasPath, os.ModePerm)
		// err = os.MkdirAll(logBasPath, 0775)
	}

	return err
}

// LogJobf info to file
func LogJobf(ctx context.Context, tpl string, args ...interface{}) {
	LogJob(ctx, fmt.Sprintf(tpl, args...))
}

// LogJob info to file
func LogJob(ctx context.Context, args ...interface{}) {
	jobMap := ctx.Value("jobParam")
	if jobMap != nil {
		jobParamMap, ok := jobMap.(map[string]map[string]interface{})["logParam"]
		if ok {
			logid, ok := jobParamMap["logId"]
			if ok {
				nowTime := time.Now()

				var buffer bytes.Buffer
				buffer.WriteString(nowTime.Format(constants.DateTimeFormat))
				buffer.WriteString("  [")

				jobName, ok := jobParamMap["jobName"]
				if ok {
					buffer.WriteString(jobName.(string))
				}
				buffer.WriteString("#")
				jobFunc, ok := jobParamMap["jobFunc"]
				if ok {
					buffer.WriteString(jobFunc.(string))
				}
				buffer.WriteString("]-[")

				jobId, ok := jobParamMap["jobId"]
				if ok {
					buffer.WriteString(fmt.Sprintf("jobId:%d", jobId.(int32)))
				}
				buffer.WriteString("]  ")
				if len(args) > 0 {
					for _, arg := range args {
						buffer.WriteString(fmt.Sprintf("%v", arg))
					}
				}
				buffer.WriteString("\r\n")

				logId := logid.(int64)
				writeLog(GetLogPath(nowTime), LogfileName(logId), buffer.String())
			}
		}
	}
}

func writeLog(logPath, logFile, msg string) error {
	if strings.Trim(logFile, " ") != "" {
		file, err := OpenDirFile(logPath, logFile)
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

// OpenDirFile open file in the dir
func OpenDirFile(logDir, fileName string) (*os.File, error) {
	fileFullPath := logDir + "/" + fileName

	// ensure log dir is created
	_, err := os.Stat(logDir)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(logBasPath, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	return OpenLogFile(fileFullPath)
}

// OpenLogFile handler
func OpenLogFile(logfile string) (*os.File, error) {
	return os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
}

// LogfileName build
func LogfileName(logId int64) string {
	return fmt.Sprintf("%d", logId) + ".log"
}

func ReadLog(logDateTim, logId int64, fromLineNum int32) (line int32, content string) {
	nowtime := time.Unix(logDateTim/1000, 0)
	fileName := GetLogPath(nowtime) + "/" + fmt.Sprintf("%d", logId) + ".log"
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
