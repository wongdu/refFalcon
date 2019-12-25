package sender

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/sdvdxl/dinghook"
)

const (
	Fastdfs_Addr = "http://47.106.192.182:9066/"
)

type DingTalk struct {
}

func (d *DingTalk) Send(token string, content string, msgType string, arr ...string) error {
	if token == "" {
		return errors.New("need dingding token")
	}

	lengthArr := len(arr)
	if lengthArr != 0 && lengthArr != 1 {
		return errors.New("variable parameter length should be zero or one")
	}
	// 发送钉钉
	ding := dinghook.NewDing(token)
	var result dinghook.Result

	if lengthArr == 0 {
		if msgType == dinghook.MsgTypeMarkdown {
			result = ding.SendMarkdown(dinghook.Markdown{Title: "告警", Content: content})
		} else {
			result = ding.SendMessage(dinghook.Message{Content: content})
		}
	} else {
		//MsgTypeActionCard用于发送自定义的告警类型
		if msgType == dinghook.MsgTypeActionCard {
			//获取具体的Counter值并分别处理
			strCounter := arr[0]
			switch strCounter {
			case "crash.log":
				packageNameIdx := strings.Index(content, "packageName=")
				if 1 >= packageNameIdx {
					return errors.New("crash log should contain endpoint")
				}
				endpoint := content[:packageNameIdx-1]

				subAddrIdx := strings.Index(content, "subAddr")
				if -1 == subAddrIdx {
					return errors.New("crash log tags should contains http address")
				}
				packageName := content[packageNameIdx+len("packageName=") : subAddrIdx-1]
				urlWithFunc := Fastdfs_Addr + content[subAddrIdx+len("subAddr="):]
				strs := strings.Split(urlWithFunc, " ")
				if 2 != len(strs) {
					return errors.New("crash log url is not nomal")
				}

				var highligthColor string
				highligthColor = "#9400D3" //DarkVoilet 深紫罗兰色
				singleURL := strs[0]
				colorEndpoint := "<font color=" + highligthColor + ">" + endpoint + "主机" + "</font>"
				notifyContent := fmt.Sprintf("%s需要处理异常日志: 应用%s出现严重错误，请点击下面链接查看完整日志", colorEndpoint, packageName)
				result = ding.SendActionCard(dinghook.ActionCard{Title: "异常日志告警", Content: notifyContent, CompleteContentURL: singleURL})
			case "crash.log.flag":
				contentIdx := strings.Index(content, "content=")
				//前面至少有主机名和+号，至少也占用两个字节
				if 1 >= contentIdx {
					return errors.New("crash log should contain endpoint")
				}
				//-1是因为需要跨过前面的+号
				endpoint := content[:contentIdx-1]
				// if !strings.HasPrefix(content, "content=") {
				// 	return errors.New("crash log tags should start with content=")
				// }
				subAddrIdx := strings.Index(content, "subAddr")
				if -1 == subAddrIdx {
					return errors.New("crash log tags should contains http address")
				}
				// crashLogContent := content[len("content=") : subAddrIdx-1]
				crashLogContent := content[contentIdx+len("content=") : subAddrIdx-1]
				urlWithFunc := Fastdfs_Addr + content[subAddrIdx+len("subAddr="):]
				strs := strings.Split(urlWithFunc, " ")
				//跳过crash.log.flag content=请点击下面链接查看完整日志,subAddr=20191030/17/13/0/c.txt 1>=1]之中的"1>=1"
				if 2 != len(strs) {
					return errors.New("crash log url is not nomal")
				}

				var highligthColor string
				highligthColor = "#9400D3" //DarkVoilet 深紫罗兰色
				singleURL := strs[0]
				colorEndpoint := "<font color=" + highligthColor + ">" + endpoint + "主机" + "</font>"
				notifyContent := fmt.Sprintf("%s需要处理异常日志: %s", colorEndpoint, crashLogContent)
				result = ding.SendActionCard(dinghook.ActionCard{Title: "异常日志告警", Content: notifyContent, CompleteContentURL: singleURL})
			case "offline.reboot":
				plusIdx := strings.Index(content, "+")
				//msg.Endpoint + "+" + msg.Tags
				//+号肯定是在第一个字节后面，因为+号前面是主机名，至少一个字节
				if 1 > plusIdx {
					return errors.New("offline reboot miss the endpoint")
				}
				endpoint := content[:plusIdx]
				offlineRebootInfoIdx := strings.Index(content, "offlineRebootInfo")
				if -1 == offlineRebootInfoIdx {
					return errors.New("offline reboot tags should contains elapse time info")
				}
				elapseInfoWithFunc := content[offlineRebootInfoIdx+len("offlineRebootInfo="):]
				strs := strings.Split(elapseInfoWithFunc, " ")
				elapseInfo := strs[0]

				var highligthColor string
				highligthColor = "#DC143C" //Crimson 猩红
				colorEndpoint := "<font color=" + highligthColor + ">" + endpoint + "主机" + "</font>"
				notifyContent := fmt.Sprintf("%s刚刚由于断网自动重启：%s后成功连上网络", colorEndpoint, elapseInfo)
				// result = ding.SendMessage(dinghook.Message{Content: notifyContent})
				//发送普通的消息，钉钉并不能看到文本高亮，所以发送markdown，Title字段并不会显示
				result = ding.SendMarkdown(dinghook.Markdown{Title: "告警", Content: notifyContent})
			default:
				log.Println("unexpeted counter")
			}
		} else {
			log.Println("unexpeted message type")
		}
	}

	log.Println(result)
	if !result.Success {
		log.Println("token:", token, " send result:", result)
		return echo.NewHTTPError(http.StatusBadRequest, result.ErrMsg)
	}

	return nil
}

func NewDingTalk() *DingTalk {
	return &DingTalk{}
}
