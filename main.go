package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func main() {
	port := flag.Int("port", 3458, "服务端口")
	listen := flag.String("listen", "", "监听地址，如 127.0.0.1 或 0.0.0.0（默认从配置读取，配置为空则为 127.0.0.1）")
	startMode := flag.Bool("start", false, "编译+启动+打开浏览器")
	flag.Parse()

	if *startMode {
		buildAndStart(*port, *listen)
		return
	}

	if err := startServer(*port, *listen); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func buildAndStart(port int, listen string) {
	exe := "aimodels.exe"
	if runtime.GOOS != "windows" {
		exe = "./aimodels"
	}

	fmt.Println("编译中...")
	ver := time.Now().Format("200601021504")
	cmd := exec.Command("go", "build", "-ldflags", "-X main.Version="+ver, "-o", exe, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("编译失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("编译完成。")

	// 检查是否已在运行
	running := false
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("powershell", "-Command",
			"Get-Process aimodels -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Id").Output()
		if len(out) > 0 {
			running = true
		}
	}

	if running {
		fmt.Println("服务已在运行。")
	} else {
		fmt.Println("启动服务...")
		args := []string{"-port", fmt.Sprintf("%d", port)}
		if listen != "" {
			args = append(args, "-listen", listen)
		}
		startCmd := exec.Command(exe, args...)
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		if err := startCmd.Start(); err != nil {
			fmt.Printf("启动失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("服务已启动。")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/admin/", port)
	fmt.Printf("\n管理后台: %s\n", url)

	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
