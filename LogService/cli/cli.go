package cli

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

func SetClientLogger(clientService string, serviceIP string, servicePort int32) {
	log.SetPrefix(fmt.Sprintf("[%v] - ", clientService))
	log.SetFlags(0)
	log.SetOutput(&clientLogger{
		url: fmt.Sprintf("%v:%v", serviceIP, servicePort),
	})
}

type clientLogger struct {
	url string
}

func (cl clientLogger) Write(data []byte) (int, error) {
	b := bytes.NewBuffer(data)
	res, err := http.Post("http://"+cl.url+"/log", "text/plain", b)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("发送日志信息失败")
	}
	return len(data), nil
}
