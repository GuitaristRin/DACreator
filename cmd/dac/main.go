// Command dac 是 DACreator 的引擎 CLI 入口。
// v3 架构：引擎负责爬取/渲染/存储/更新，GUI（gui/）通过子进程与本入口通信。
package main

import (
	"fmt"
	"os"
)

// version 由构建脚本通过 ldflags 注入：-X main.version=x.y.z
var version = "dev"

func main() {
	fmt.Printf("DACreator 引擎 %s\n", version)
	os.Exit(0)
}
