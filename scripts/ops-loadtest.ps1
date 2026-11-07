#Requires -Version 5.1
<#
.SYNOPSIS
    Planstar 运维端点压测（Windows PowerShell 版，等价于 ops-loadtest.sh）。

.DESCRIPTION
    运行 TestOpsEndpointsLoad：100 goroutines × 200 请求，覆盖 /healthz、
    /readyz，并校验 /metrics 无令牌 401。输出 QPS/P50/P95/P99/错误率。

    用法（任意目录）：
        powershell -ExecutionPolicy Bypass -File scripts\ops-loadtest.ps1
#>

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location (Join-Path $RepoRoot 'planstar')

& go test ./internal/server -run TestOpsEndpointsLoad -count=1 -v
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
exit 0
