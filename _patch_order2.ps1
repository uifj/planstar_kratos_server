$enc = New-Object System.Text.UTF8Encoding($false)
Get-ChildItem service\order\internal -Recurse -Filter *.go | ForEach-Object {
    $t = [System.IO.File]::ReadAllText($_.FullName)
    $o = $t
    $t = $t -replace '"github\.com/go-kratos/kratos/v2/log"', '"log/slog"'
    $t = $t -replace 'logger log\.Logger', 'logger *slog.Logger'
    $t = $t -replace '\*log\.Helper', '*slog.Logger'
    $t = $t -replace 'log\.NewHelper\(logger\)', 'logger'
    if ($t -ne $o) {
        [System.IO.File]::WriteAllText($_.FullName, $t, $enc)
        Write-Output ("xformed: " + $_.Name)
    }
}
