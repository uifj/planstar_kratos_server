package main

import (
	"flag"
	"os"
)

var (
	Name    = "planstar.api"
	Version = "planstar.api.v1"
	// flagconf 是配置路径。
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf configs/config.yaml")
}

// main 只负责解析参数与最终退出码。全部装配与资源生命周期在 wiring.go 的 run() 中，
// run 返回前其 defer 链（tracing / MySQL / MQ / 后台任务）已按 LIFO 执行完毕。
func main() {
	flag.Parse()
	if err := run(); err != nil {
		os.Exit(1)
	}
}
