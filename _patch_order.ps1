$enc = New-Object System.Text.UTF8Encoding($false)
$files = Get-ChildItem service\order -Recurse -Filter *.go
$n = 0
foreach ($f in $files) {
    $t = [System.IO.File]::ReadAllText($f.FullName)
    $o = $t
    $t = $t -replace 'github\.com/go-kratos/kratos/v2/(errors|transport|config|middleware|registry)', 'github.com/go-kratos/kratos/v3/$1'
    if ($t -ne $o) {
        [System.IO.File]::WriteAllText($f.FullName, $t, $enc)
        $n++
        Write-Output ("patched: " + $f.FullName)
    }
}
Write-Output ("total " + $n)
