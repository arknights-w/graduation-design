package main

import (
	"cli"
	"cli/api/airfone"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	registryURL             = "0.0.0.0:9000"      // 服务发现中心的地址
	fileURL                 = "./distributed.log" // 打印日志的文件路径
	localhost               = "0.0.0.0"           // 服务地址
	port        int32       = 5000                // 服务端口
	logger      *log.Logger                       // log 实体
)

type fileLog struct {
	FileURL string // 写入的文件路径
}

func (fl *fileLog) Write(data []byte) (int, error) {
	// 打开 日志文件
	f, err := os.OpenFile(string(fl.FileURL), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return 0, err
	}
	// 函数结束默认关闭文件资源
	defer f.Close()
	// 将 日志消息 写入 文件 并返回
	return f.Write(data)
}

// 初始化全局的 logger
func InitStaticLogger(distination string) {
	file_log := &fileLog{
		FileURL: distination,
	}
	logger = log.New(file_log, "mylog ---- ", log.LstdFlags)
}

// 将数据打印在logger文件中
func write(message string) {
	logger.Printf("%v\n", message)
}

func DoLog(ctx *gin.Context) {
	var (
		bytes []byte
		err   error
	)
	if bytes, err = ctx.GetRawData(); err != nil {
		ctx.Status(http.StatusBadRequest)
	}
	fmt.Printf("msg.Msg: %s\n", bytes)
	write(string(bytes))
	ctx.Status(http.StatusOK)
}

func main() {
	var (
		url = fmt.Sprintf("%v:%v", localhost, port)
	)
	// 初始化全局 logger
	InitStaticLogger(fileURL)
	// new 注册中心客户端
	client, err := cli.NewCli(registryURL)
	if err != nil {
		log.Fatal("new register client fail: ",err)
	}
	// 配置注册
	if err := client.Register(&cli.Config{
		Topic: "log",
		IP:    localhost,
		Port:  port,
		Schema: []*airfone.Schema{
			{
				Title:   "kk123",
				Content: "321kk",
			},
		},
	}); err != nil {
		log.Fatal("register to center fail: ",err)
	}

	// 日志服务器
	router := gin.Default()
	router.POST("/log", DoLog)
	server := http.Server{
		Addr:    url,
		Handler: router,
	}
	go func() {
		// 服务连接
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	// 向注册中心注销
	if err = client.Logout();err != nil{
		log.Fatal("logout failed: ",err)
	}
	// 关闭服务器
	log.Println("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
