#Requires -Version 5.1
<#
.SYNOPSIS
    Planstar 本地烟雾测试（Windows PowerShell 版，等价于 smoke-local.sh）。

.DESCRIPTION
    前置：cd planstar; Copy-Item .env.example .env; docker compose up -d --build
    用法：
        powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1
        $env:PS_API_BASE='http://127.0.0.1:8098'; $env:PS_ADMIN_BASE='http://127.0.0.1:9108'
        powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1

    检查项：
      1. GET  <admin>/healthz                            -> 200
      2. GET  <api>/v1/activities?pageIndex=1&pageSize=1 -> 200 且 JSON 可解析
      3. POST <api>/v3/user/sendLoginCode                -> 不得返回 5xx
    任一失败即以非 0 退出，并输出具体原因。
#>

$ErrorActionPreference = 'Stop'

$ApiBase = if ($env:PS_API_BASE) { $env:PS_API_BASE } else { 'http://127.0.0.1:8098' }
$AdminBase = if ($env:PS_ADMIN_BASE) { $env:PS_ADMIN_BASE } else { 'http://127.0.0.1:9108' }

$script:Failures = 0

function Write-Pass {
    param([string]$Message)
    Write-Host "OK:   $Message" -ForegroundColor Green
}

function Write-Fail {
    param([string]$Message)
    Write-Host "FAIL: $Message" -ForegroundColor Red
    $script:Failures++
}

function Invoke-SmokeRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Url,
        [string]$Body
    )

    $result = [pscustomobject]@{ Status = 0; Content = ''; Error = $null }

    $params = @{
        Uri             = $Url
        Method          = $Method
        UseBasicParsing = $true
        TimeoutSec      = 10
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

Write-Host "==> 1/3 运维端口存活探针：$AdminBase/healthz"
$r = Invoke-SmokeRequest -Method 'GET' -Url "$AdminBase/healthz"
if ($r.Error) {
    Write-Fail "无法连接 $AdminBase/healthz（容器未启动或 9108 未映射）：$($r.Error)"
}
elseif ($r.Status -ne 200) {
    Write-Fail "GET /healthz 期望 200，实际 $($r.Status)"
}
else {
    Write-Pass "GET /healthz -> 200"
}

Write-Host "==> 2/3 演出目录：$ApiBase/v1/activities?pageIndex=1&pageSize=1"
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
        $preview = $r.Content
        if ($preview.Length -gt 80) { $preview = $preview.Substring(0, 80) }
        Write-Fail "GET /v1/activities 响应不是可解析的 JSON（前 80 字符：$preview）"
    }
}

Write-Host "==> 3/3 发码接口：POST $ApiBase/v3/user/sendLoginCode"
$r = Invoke-SmokeRequest -Method 'POST' -Url "$ApiBase/v3/user/sendLoginCode" -Body '{"ua":"13800000001"}'
if ($r.Error) {
    Write-Fail "无法连接 $ApiBase/v3/user/sendLoginCode：$($r.Error)"
}
elseif ($r.Status -ge 500) {
    $preview = $r.Content
    if ($preview.Length -gt 200) { $preview = $preview.Substring(0, 200) }
    Write-Fail "POST /v3/user/sendLoginCode 返回 5xx（$($r.Status)）：$preview"
}
elseif ($r.Status -ge 200) {
    Write-Pass "POST /v3/user/sendLoginCode -> $($r.Status)（非 5xx）"
}
else {
    Write-Fail "POST /v3/user/sendLoginCode 返回异常状态码：$($r.Status)"
}

if ($script:Failures -ne 0) {
    Write-Host ""
    Write-Host "烟雾测试失败：$($script:Failures) 项未通过。" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "烟雾测试通过：3/3。" -ForegroundColor Green
exit 0
