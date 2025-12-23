package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alireza0/s-ui/app"
	"github.com/alireza0/s-ui/cmd"
)

func runApp() {
	a := app.NewApp()

	err := a.Init()
	if err != nil {
		log.Fatal(err)
	}

	err = a.Start()
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	// Trap shutdown signals
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			a.RestartApp()
		default:
			a.Stop()
			return
		}
	}
}

func main() {
	// 解析命令行参数
	if !cmd.ParseFlags() {
		return
	}

	// 检查是否有子命令
	args := flag.Args()
	if len(args) > 0 {
		// 有子命令，交给 ParseCmd 处理
		cmd.ParseCmd()
		return
	}

	// 没有子命令，启动应用
	runApp()
}
