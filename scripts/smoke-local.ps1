#Requires -Version 5.1
<#
.SYNOPSIS
    Planstar 本地烟雾测试（Windows PowerShell 版，等价于 smoke-local.sh）。

.DESCRIPTION
    前置：cd planstar; Copy-Item .env.example .env; docker compose up -d --build
    用法：
        powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1
        $env:PLANSTAR_OPS_TOKEN='<token>'
        powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1

    检查项：
      1. GET  <admin>/healthz                            -> 200 且 status=ok
      2. GET  <admin>/readyz                             -> 200/503 且含 checks/ready
      3. GET  <admin>/metrics（无令牌）                  -> 401
      4. GET  <admin>/metrics（Bearer）                  -> 200 且含 build_info
      5. GET  <api>/v1/activities?pageIndex=1&pageSize=1 -> 200 且 JSON 可解析
      6. POST <api>/v3/user/sendLoginCode                -> 不得返回 5xx
    令牌为空时跳过 3/4 并给出提示。
    任一失败即以非 0 退出，并输出具体原因。
#>

$ErrorActionPreference = 'Stop'

$ApiBase = if ($env:PS_API_BASE) { $env:PS_API_BASE } else { 'http://127.0.0.1:8098' }
$AdminBase = if ($env:PS_ADMIN_BASE) { $env:PS_ADMIN_BASE } else { 'http://127.0.0.1:9108' }
$OpsToken = if ($env:PLANSTAR_OPS_TOKEN) { $env:PLANSTAR_OPS_TOKEN } else { '' }

$script:Failures = 0
$script:Passed = 0
$script:Skipped = 0

function Write-Pass {
    param([string]$Message)
    Write-Host "OK:   $Message" -ForegroundColor Green
    $script:Passed++
}

function Write-Fail {
    param([string]$Message)
    Write-Host "FAIL: $Message" -ForegroundColor Red
    $script:Failures++
}

function Write-Skip {
    param([string]$Message)
    Write-Host "SKIP: $Message" -ForegroundColor Yellow
    $script:Skipped++
}

function Invoke-SmokeRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Url,
        [string]$Body,
        [hashtable]$Headers = @{}
    )

    $result = [pscustomobject]@{ Status = 0; Content = ''; Error = $null }

    $params = @{
        Uri             = $Url
        Method          = $Method
        UseBasicParsing = $true
        TimeoutSec      = 10
    }
    if ($Headers.Count -gt 0) {
        $params['Headers'] = $Headers
    }
    if ($Body) {
        $params['ContentType'] = 'application/json'
        $params['Body'] = $Body
    }

    try {
        $resp = Invoke-WebRequest @params
        $result.Status = [int]$resp.StatusCode
        $result.Content = [string]$resp.Content
    }
    catch {
        $ex = $_.Exception
        $webResp = $ex.Response
        if ($null -ne $webResp) {
            try { $result.Status = [int]$webResp.StatusCode } catch { $result.Status = 0 }
            try {
                $stream = $webResp.GetResponseStream()
                if ($null -ne $stream) {
                    $reader = New-Object System.IO.StreamReader($stream)
                    $result.Content = $reader.ReadToEnd()
                    $reader.Dispose()
                }
            }
            catch {
                # 非 HttpWebResponse 实现（如 PowerShell 7）无法读取错误响应体，忽略。
            }
        }
        else {
            $result.Error = $ex.Message
        }
    }

    return $result
}

function Get-Preview {
    param([string]$Text, [int]$Length = 200)
    if ([string]::IsNullOrEmpty($Text)) { return '' }
    if ($Text.Length -gt $Length) { return $Text.Substring(0, $Length) }
    return $Text
}

Write-Host "==> 1/6 运维存活探针：$AdminBase/healthz"
$r = Invoke-SmokeRequest -Method 'GET' -Url "$AdminBase/healthz"
if ($r.Error) {
    Write-Fail "无法连接 $AdminBase/healthz（容器未启动或 9108 未映射）：$($r.Error)"
}
elseif ($r.Status -ne 200) {
    Write-Fail "GET /healthz 期望 200，实际 $($r.Status)"
}
else {
    try {
        $json = $r.Content | ConvertFrom-Json
        if ($json.status -ne 'ok') {
            Write-Fail "GET /healthz 响应 status 不为 ok：$(Get-Preview $r.Content)"
        }
        else {
            Write-Pass "GET /healthz -> 200 且 status=ok"
        }
    }
    catch {
        Write-Fail "GET /healthz 响应不是可解析的 JSON：$(Get-Preview $r.Content)"
    }
}

Write-Host "==> 2/6 运维就绪探针：$AdminBase/readyz"
$r = Invoke-SmokeRequest -Method 'GET' -Url "$AdminBase/readyz"
if ($r.Error) {
    Write-Fail "无法连接 $AdminBase/readyz：$($r.Error)"
}
elseif ($r.Status -ne 200 -and $r.Status -ne 503) {
    Write-Fail "GET /readyz 期望 200 或 503，实际 $($r.Status)"
}
else {
    try {
        $json = $r.Content | ConvertFrom-Json
        $names = @($json.PSObject.Properties.Name)
        if (-not ($names -contains 'checks') -or -not ($names -contains 'ready')) {
            Write-Fail "GET /readyz 响应缺少 checks/ready 字段：$(Get-Preview $r.Content 300)"
        }
        elseif ($r.Status -eq 503) {
            $failed = @($json.checks | Where-Object { $_.status -eq 'fail' } | ForEach-Object { $_.name })
            $failedText = if ($failed.Count -gt 0) { $failed -join ', ' } else { '(未解析到检查名)' }
            Write-Fail "GET /readyz -> 503，失败检查：$failedText"
        }
        else {
            Write-Pass "GET /readyz -> 200 且包含 checks/ready"
        }
    }
    catch {
        Write-Fail "GET /readyz 响应不是可解析的 JSON：$(Get-Preview $r.Content)"
    }
}

if ([string]::IsNullOrEmpty($OpsToken)) {
    Write-Host "==> 3/6 /metrics 鉴权（无令牌）"
    Write-Skip "PLANSTAR_OPS_TOKEN 未设置；服务端鉴权关闭，跳过 /metrics 鉴权检查"
    Write-Host "==> 4/6 /metrics 构建信息（Bearer）"
    Write-Skip "PLANSTAR_OPS_TOKEN 未设置，跳过 /metrics Bearer 抓取"
}
else {
    Write-Host "==> 3/6 /metrics 鉴权（无令牌）：$AdminBase/metrics"
    $r = Invoke-SmokeRequest -Method 'GET' -Url "$AdminBase/metrics"
    if ($r.Error) {
        Write-Fail "无法连接 $AdminBase/metrics：$($r.Error)"
    }
    elseif ($r.Status -ne 401) {
        Write-Fail "GET /metrics 无令牌期望 401，实际 $($r.Status)"
    }
    else {
        Write-Pass "GET /metrics 无令牌 -> 401"
    }

    Write-Host "==> 4/6 /metrics 构建信息（Bearer）：$AdminBase/metrics"
    $headers = @{ Authorization = "Bearer $OpsToken" }
    $r = Invoke-SmokeRequest -Method 'GET' -Url "$AdminBase/metrics" -Headers $headers
    if ($r.Error) {
        Write-Fail "无法连接 $AdminBase/metrics（Bearer）：$($r.Error)"
    }
    elseif ($r.Status -ne 200) {
        Write-Fail "GET /metrics 带 Bearer 期望 200，实际 $($r.Status)"
    }
    elseif ($r.Content -notmatch 'planstar_build_info') {
        Write-Fail "GET /metrics 响应缺少 planstar_build_info"
    }
    else {
        Write-Pass "GET /metrics 带 Bearer -> 200 且包含 planstar_build_info"
    }
}

Write-Host "==> 5/6 演出目录：$ApiBase/v1/activities?pageIndex=1&pageSize=1"
$r = Invoke-SmokeRequest -Method 'GET' -Url "$ApiBase/v1/activities?pageIndex=1&pageSize=1"
if ($r.Error) {
    Write-Fail "无法连接 $ApiBase/v1/activities（容器未启动或 8098 未映射）：$($r.Error)"
}
elseif ($r.Status -ne 200) {
    Write-Fail "GET /v1/activities 期望 200，实际 $($r.Status)"
}
elseif ([string]::IsNullOrWhiteSpace($r.Content)) {
    Write-Fail "GET /v1/activities 返回空响应体"
}
else {
    try {
        $null = $r.Content | ConvertFrom-Json
        Write-Pass "GET /v1/activities -> 200 且 JSON 可解析"
    }
    catch {
        Write-Fail "GET /v1/activities 响应不是可解析的 JSON（前 80 字符：$(Get-Preview $r.Content 80)）"
    }
}

Write-Host "==> 6/6 发码接口：POST $ApiBase/v3/user/sendLoginCode"
$r = Invoke-SmokeRequest -Method 'POST' -Url "$ApiBase/v3/user/sendLoginCode" -Body '{"ua":"13800000001"}'
if ($r.Error) {
    Write-Fail "无法连接 $ApiBase/v3/user/sendLoginCode：$($r.Error)"
}
elseif ($r.Status -ge 500) {
    Write-Fail "POST /v3/user/sendLoginCode 返回 5xx（$($r.Status)）：$(Get-Preview $r.Content)"
}
elseif ($r.Status -ge 200) {
    Write-Pass "POST /v3/user/sendLoginCode -> $($r.Status)（非 5xx）"
}
else {
    Write-Fail "POST /v3/user/sendLoginCode 返回异常状态码：$($r.Status)"
}

if ($script:Failures -ne 0) {
    Write-Host ""
    Write-Host "烟雾测试失败：$($script:Failures) 项未通过（通过 $($script:Passed)，跳过 $($script:Skipped)）。" -ForegroundColor Red
    exit 1
}

Write-Host ""
if ($script:Skipped -ne 0) {
    Write-Host "烟雾测试通过：$($script:Passed)/6 项通过，$($script:Skipped) 项跳过。" -ForegroundColor Green
}
else {
    Write-Host "烟雾测试通过：6/6。" -ForegroundColor Green
}
exit 0
