package watch

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/agent/funcs"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
	"github.com/toolkits_/file"
	"github.com/toolkits_/sys"
)

var (
	// timeout for watcher event
	checkFileHandleTimeOut  = time.Second * 10
	uploadFileHandleTimeOut = time.Second * 10
	watchLoopSleepTimeOut   = time.Second * 10
	cUploadFile             chan string
)

const (
	UploadFile_Chan_Size       = 5
	Crash_Log_File_Rename_Time = 5
)

type cbFuncType func(string)

var (
	cbFunc cbFuncType
)

func init() {
	// cUploadFile = make(chan string)
	cUploadFile = make(chan string, UploadFile_Chan_Size)
}

func SetCbFunc(cb cbFuncType) {
	cbFunc = cb
}

func NotifyIncrementFile(fileName string) {
	go func(file_name string) {
		cUploadFile <- file_name
	}(fileName)
}

func ProWatchDir() {
	// no need send flag previously
	// sendCrashLogFlag(false, "")
	for {
		select {
		case <-time.After(uploadFileHandleTimeOut):
		//do nothing
		case fileName, ok := <-cUploadFile:
			if !ok {
				log.Println("chan upload file closed")
				break
			}

			//go uploadFile(fileName)
			go uploadIncrementFile(fileName)
		}
	}
	log.Println("loop upload file over,check whether this is normal")
}

//新的上报fatal log方法，而非监控/data下system子目录的dropbox目录
func uploadIncrementFile(filePath string) {
	if "" == filePath {
		log.Println("upload increment file name can not be empty")
		return
	}

	filePathAbs, _ := filepath.Abs(filePath)
	fileInfo, fileErr := os.Stat(filePathAbs)
	if fileErr != nil {
		log.Println("get increment file stat error")
		return
	}

	if true == fileInfo.IsDir() || fileInfo.Size() <= 0 {
		log.Println("increment file is not a regular file or size equal zero")
		return
	}

	packageName := getPackageName(filePath)

	//curl上传增量文件到阿里云，并获取地址
	var bs []byte
	var cmdline string
	cmdline = fmt.Sprintf("curl -F file=@%s http://47.106.192.182:9067/group1/upload", filePathAbs)
	bs, err := sys.CmdOutBytes("/system/bin/sh", "-c", cmdline)
	// bs, err := sys.CmdOutBytes("/system/xbin/su", "-c", cmdline)
	if err != nil {
		log.Println("upload increment file failed when curl: ", err)
		return
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))

	line, err := file.ReadLine(reader)
	if err != nil {
		log.Println("upload increment file failed when curl: ", err)
		return
	}

	strAddr := string(line)
	idx := strings.Index(strAddr, "default")
	if -1 == idx {
		log.Println("upload increment file response content is wrong")
		return
	}
	subAddr := strAddr[idx+len("default")+1:]
	log.Println("get upload increment file sub addr is: ", subAddr)

	tags := fmt.Sprintf("packageName=%s,subAddr=%s", packageName, subAddr)
	sendCrashLog(true, tags)
}

func sendCrashLog(bFlag bool, tags string) {
	var mvs []*model.MetricValue
	if bFlag {
		mvs = []*model.MetricValue{funcs.GaugeValue("crash.log", 1, tags)}
	} else {
		mvs = []*model.MetricValue{funcs.GaugeValue("crash.log", 0)}
	}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)
}

func uploadFile(filePath string) {
	if "" == filePath {
		log.Println("upload file name can not be empty")
		return
	}

	//其实从android层传递过来的已经是文件绝对路径
	if strings.HasSuffix(filePath, ".tmp") {
		filePath = getRealFilePath(filePath)
	}
	if "" == filePath {
		log.Println("get real upload file name failed")
		return
	}

	filePathAbs, _ := filepath.Abs(filePath)
	fileInfo, fileErr := os.Stat(filePathAbs)
	if fileErr != nil {
		log.Println("get upload file stat error")
		return
	}

	if true == fileInfo.IsDir() || fileInfo.Size() <= 0 {
		log.Println("upload file is not a regular file or size equal zero")
		return
	}

	crashLogContent := getCrashLogContent(filePathAbs)
	var bs []byte
	var cmdline string
	/*cmdline = fmt.Sprintf("chmod 666 %s", filePathAbs)
	bs, err := sys.CmdOutBytes("/system/bin/sh", "-c", cmdline)
	if err != nil {
		log.Println("chmod file failed ", err)
		return
	}*/
	if cbFunc != nil {
		cbFunc(filePathAbs)
	}

	cmdline = fmt.Sprintf("curl -F file=@%s http://47.106.192.182:9067/group1/upload", filePathAbs)
	bs, err := sys.CmdOutBytes("/system/bin/sh", "-c", cmdline)
	// bs, err := sys.CmdOutBytes("/system/xbin/su", "-c", cmdline)
	if err != nil {
		log.Println("upload file failed when curl: ", err)
		return
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))

	line, err := file.ReadLine(reader)
	if err != nil {
		log.Println("upload file failed when curl: ", err)
		return
	}

	strAddr := string(line)
	idx := strings.Index(strAddr, "default")
	if -1 == idx {
		log.Println("upload file response content is wrong")
		return
	}
	subAddr := strAddr[idx+len("default")+1:]
	log.Println("get upload sub addr is: ", subAddr)

	// mvs := []*model.MetricValue{GaugeValue("crash.log.flag", 1, tag)}
	tags := fmt.Sprintf("content=%s,subAddr=%s", crashLogContent, subAddr)
	sendCrashLogFlag(true, tags)
}

func sendCrashLogFlag(bFlag bool, tags string) {
	var mvs []*model.MetricValue
	if bFlag {
		mvs = []*model.MetricValue{funcs.GaugeValue("crash.log.flag", 1, tags)}
	} else {
		mvs = []*model.MetricValue{funcs.GaugeValue("crash.log.flag", 0)}
	}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)
}

func getPackageName(filePath string) string {
	if "" == filePath {
		return ""
	}

	lastSepIdx := strings.LastIndex(filePath, "/")
	lastAtIdx := strings.LastIndex(filePath, "@")
	if lastSepIdx == -1 || lastAtIdx == -1 || lastSepIdx > lastAtIdx {
		return ""
	}

	return filePath[lastSepIdx+1 : lastAtIdx]
}

func getCrashLogContent(filePath string) string {
	//no need to judge the file path
	if !strings.HasSuffix(filePath, ".txt") {
		return fmt.Sprintf("非txt崩溃日志，请点击下面链接下载查看")
	}

	var bs []byte
	var cmdline string
	cmdline = fmt.Sprintf("cat %s |grep Package |tail -n 1", filePath)
	bs, err := sys.CmdOutBytes("/system/bin/sh", "-c", cmdline)
	if err != nil {
		log.Println("find the Package name failed: ", err)
		return fmt.Sprintf("请点击下面链接查看完整日志")
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))

	line, err := file.ReadLine(reader)
	if "" == string(line) {
		return fmt.Sprintf("请点击下面链接查看完整日志")
	}
	results := strings.Split(string(line), ":")
	if 2 != len(results) {
		return fmt.Sprintf("请点击下面链接查看完整日志")
	}
	return fmt.Sprintf("%s异常，请点击下面链接下载查看完整日志", results[1])
}

func getRealFilePath(fileAbsPath string) (realFilePath string) {
	if "" == fileAbsPath {
		log.Println("can not get real path ,cause file absolute path is empty")
		return
	}

	idx := strings.LastIndex(fileAbsPath, "/")
	if idx < 0 {
		log.Printf("%s is not regular file path for real path ", fileAbsPath)
		return
	}
	fileName := fileAbsPath[idx+1:]
	if !strings.HasPrefix(fileName, "drop") {
		return fileAbsPath
	}

	dirPath := fileAbsPath[:idx+1]
	info, err := os.Stat(dirPath)
	if err != nil {
		return
	}
	if false == info.IsDir() {
		return
	}

	rd, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return
	}

	var mapFileTime map[int64]string
	mapFileTime = make(map[int64]string, len(rd))
	for _, fi := range rd {
		if fi.IsDir() {
			continue
		}
		//fileAbsPath, _ := filepath.Abs(fi.Name())
		//modTime, err := getFileModTime(fileAbsPath)
		//modTime, err := getFileModifyTime(fileAbsPath)

		//modTime, err := getFileModTime(dirPath + fi.Name())
		modTime, err := getFileModifyTime(dirPath + fi.Name())
		if err != nil {
			continue
		}
		mapFileTime[modTime] = fi.Name()
	}

	// benchMarkTime := 0
	var benchMarkTime int64
	realFileName := ""
	for k, v := range mapFileTime {
		if k > benchMarkTime {
			benchMarkTime = k
			realFileName = v
		}
	}

	if "" == realFileName {
		return
	}

	renameTime := time.Now().Unix() - benchMarkTime
	log.Println("rename crash log file time", renameTime)
	if renameTime < Crash_Log_File_Rename_Time {
		realFilePath = dirPath + realFileName
	}

	return
}

func getFileModifyTime(filePathAbs string) (int64, error) {
	// log.Println("get file modify time", filePathAbs)
	fileInfo, fileErr := os.Stat(filePathAbs)
	if fileErr != nil {
		log.Println("get file stat error")
		return 0, errors.New("get file stat error")
	}
	return fileInfo.ModTime().Unix(), nil
}

func getFileModTime(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Println("open file error")
		//return time.Now().Unix()
		return 0, errors.New("open file error")
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Println("stat fileinfo error")
		return 0, errors.New("stat fileinfo error")
	}

	return fi.ModTime().Unix(), nil
}
