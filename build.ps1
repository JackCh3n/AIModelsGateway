<#
.SYNOPSIS
    AI Models Gateway 编译启动脚本
.DESCRIPTION
    先杀掉正在运行的 aimodels 进程，然后重新编译并启动
#>

$ErrorActionPreference = "Stop"
$exeName = "aimodels.exe"
$port = 3458

# 1. 杀掉旧进程
Write-Host "[1/4] 检查并停止旧进程..." -ForegroundColor Yellow
$procs = Get-Process -Name "aimodels" -ErrorAction SilentlyContinue
if ($procs) {
    Stop-Process -Name "aimodels" -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 500
    Write-Host "  旧进程已停止" -ForegroundColor Green
} else {
    Write-Host "  无运行中的进程" -ForegroundColor Gray
}

# 2. 释放端口
Write-Host "[2/4] 检查端口 $port..." -ForegroundColor Yellow
$conn = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
if ($conn) {
    $procIds = $conn | Select-Object -ExpandProperty OwningProcess -Unique
    foreach ($pid in $procIds) {
        if ($pid -ne 0) {
            Write-Host "  端口被 PID=$pid 占用，强制释放" -ForegroundColor Gray
            Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
        }
    }
    Start-Sleep -Milliseconds 500
    Write-Host "  端口已释放" -ForegroundColor Green
} else {
    Write-Host "  端口 $port 空闲" -ForegroundColor Gray
}

# 3. 编译
Write-Host "[3/4] 编译中..." -ForegroundColor Yellow
$ver = Get-Date -Format "yyyyMMddHHmm"
go build -ldflags "-X main.Version=$ver" -o $exeName .
if ($LASTEXITCODE -ne 0) {
    Write-Host "  编译失败!" -ForegroundColor Red
    exit 1
}
Write-Host "  编译成功 -> $exeName (v$ver)" -ForegroundColor Green

# 4. 启动
Write-Host "[4/4] 启动服务..." -ForegroundColor Yellow
Start-Process -FilePath ".\$exeName" -ArgumentList "-port", $port -WindowStyle Normal
Start-Sleep -Seconds 1

# 验证启动
$running = Get-Process -Name "aimodels" -ErrorAction SilentlyContinue
if ($running) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  AI Models Gateway 已启动" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  管理后台: http://127.0.0.1:$port/admin/" -ForegroundColor White
    Write-Host "  OpenAI:  http://127.0.0.1:$port/v1/chat/completions" -ForegroundColor White
    Write-Host "  Anthropic: http://127.0.0.1:$port/v1/messages" -ForegroundColor White
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""

    Start-Process "rundll32" -ArgumentList "url.dll,FileProtocolHandler", "http://127.0.0.1:$port/admin/"
} else {
    Write-Host "  启动失败!" -ForegroundColor Red
    exit 1
}
