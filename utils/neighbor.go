package utils

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/fatih/color"
)

// 用于检查给定的主机和端口是否可达。
// 使用 net.DialTimeout 函数来建立 TCP 连接，并设置了连接的超时时间为 1 秒。

func IsFoundHost(host string, port uint16) bool {
	target := fmt.Sprintf("%s:%d", host, port)

	_, err := net.DialTimeout("tcp", target, 1*time.Second)
	if err != nil {
		color.Red("[IsFoundHost Error]%s %v\n", target, err)
		return false
	} else {
		color.Green("[IsFoundHost Found]%s %v\n", target, err)
		return true
	}

}

// 该正则表达式用于匹配IP地址的模式。
var PATTERN = regexp.MustCompile(`((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?\.){3})(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)

func FindNeighbors(myHost string, myPort uint16, startIp uint8, endIp uint8, startPort uint16, endPort uint16) []string {

	address := fmt.Sprintf("%s:%d", myHost, myPort)

	m := PATTERN.FindStringSubmatch(myHost)
	fmt.Println("m:", m)
	if m == nil {
		return nil
	}
	prefixHost := m[1]                     //127.0.0
	lastIp, _ := strconv.Atoi(m[len(m)-1]) //IP地址最后一位
	neighbors := make([]string, 0)

	for port := startPort; port <= endPort; port += 1 {
		for ip := startIp; ip <= endIp; ip += 1 {
			guessHost := fmt.Sprintf("%s%d", prefixHost, lastIp+int(ip))
			guessTarget := fmt.Sprintf("%s:%d", guessHost, port)
			if guessTarget != address && IsFoundHost(guessHost, port) {
				neighbors = append(neighbors, guessTarget)
			}
		}
	}
	return neighbors
}
func GetHost() string {

	return "127.0.0.1"
}

//连手机热点有时候会找不到邻居
/*func GetHost() string {
	// 获取本地主机名
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("无法获取主机名：", err)
		return "127.0.0.1"
	}

	// 获取主机的网络接口列表
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("无法获取网络接口列表：", err)
		return "127.0.0.1"
	}

	// 遍历网络接口列表，查找非回环接口的有效 IP 地址
	for _, iface := range interfaces {
		// 排除回环接口和无效接口
		if iface.Flags&net.FlagUp != 0 {
			addresses, err := iface.Addrs()
			if err != nil {
				fmt.Println("无法获取接口地址：", err)
				continue
			}

			// 遍历接口地址，查找有效 IP 地址
			for _, addr := range addresses {
				ipNet, ok := addr.(*net.IPNet)
				if ok {
					// 排除 IPv6 地址和非全局单播地址
					if ipNet.IP.To4() != nil && ipNet.IP.IsGlobalUnicast() {
						fmt.Printf("主机名: %s，有效 IPv4 地址: %s\n", hostname, ipNet.IP.String())
						return ipNet.IP.String()
					}
				}
			}
		}
	}
	return "127.0.0.99"
}*/
