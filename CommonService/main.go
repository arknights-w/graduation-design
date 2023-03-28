package main

import (
	register "cli"
	"context"
	"fmt"
	"log"
	"logserv/cli"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	registryURL       = "0.0.0.0:9000" // 服务发现中心的地址
	localhost         = "0.0.0.0"      // 服务地址
	port        int32 = 3000           // 服务端口
)

func main() {
	var (
		url = fmt.Sprintf("%v:%v", localhost, port)
	)
	client, err := register.NewCli(registryURL)
	if err != nil {
		log.Fatal("new register client fail: ", err)
	}
	// 配置注册
	if err := client.Register(&register.Config{
		Topic:  "common",
		IP:     localhost,
		Port:   port,
		Relies: []string{"log"},
	}); err != nil {
		log.Fatal("register to center fail: ", err)
	}
	// 配置日志服务
	logcfg := client.Relies["log"]
	cli.SetClientLogger("common_serv", logcfg.Ip, logcfg.Port)

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"account": "账号",
			"pwd":     "密码",
			"submit":  "提交",
		})
	})
	router.POST("/submit", func(ctx *gin.Context) {
		type submit struct {
			Account  string
			Password string
		}
		var req submit
		if err := ctx.BindJSON(&req); err != nil {
			fmt.Printf("err: %v\n", err)
			ctx.Status(http.StatusBadRequest)
		} else {
			log.Print(req)
			ctx.JSON(http.StatusOK, gin.H{
				"code":   123321,
				"status": "芜湖起飞",
				"acct":   req.Account,
				"pwd":    req.Password,
			})
		}
	})
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
	if err = client.Logout(); err != nil {
		log.Fatal("logout failed: ", err)
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
