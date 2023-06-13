package main

import (
	"flag"
	"github.com/fatih/color"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	color.HiGreen("====================")
	color.Red("====启动区块链节点=====")
	color.HiGreen("====================")

	log.SetPrefix("Blockchain: ")
}

func main() {

	//添加-port 8888（例）参数  不写就是默认5000
	port := flag.Uint("port", 5000, "TCP Port Number for Blockchain Server")
	flag.Parse()
	//实例化一个节点服务器
	app := NewBlockchainServer(uint16(*port))
	// 捕获终止信号通道
	signalCh := make(chan os.Signal, 1)
	//将SIGINT（终止进程的中断信号）和SIGTERM（终止进程的终止信号）连接到signalCh通道
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("Server is running on port %d", app.Port())
		app.Run()
	}()
	// 阻塞操作，它会等待从signalCh通道接收到终止信号，一旦接收到终止信号程序将往下执行。
	<-signalCh

	// 执行关闭操作和保存缓存数据到本地
	app.Close()
}
