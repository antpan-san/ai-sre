# PowerShell自动构建脚本
# 监听src目录的文件变化，自动执行npm run build命令

Write-Host "=== FleetPilot前端自动构建脚本 ===" -ForegroundColor Green
Write-Host "监听目录: $PSScriptRoot\src" -ForegroundColor Cyan
Write-Host "按 Ctrl+C 停止脚本" -ForegroundColor Yellow

# 设置文件系统监视器
$watcher = New-Object System.IO.FileSystemWatcher
$watcher.Path = "$PSScriptRoot\src"
$watcher.Filter = "*.*"
$watcher.IncludeSubdirectories = $true
$watcher.EnableRaisingEvents = $true

# 设置要监听的文件类型
$allowedExtensions = ".vue", ".ts", ".js", ".css", ".scss", ".json"

# 防抖时间（毫秒），避免频繁构建
$debounceTime = 1000
$lastBuildTime = 0

# 构建函数
function BuildProject {
    $currentTime = [DateTime]::Now.Ticks / 10000
    if ($currentTime - $lastBuildTime -lt $debounceTime) {
        return
    }
    
    $lastBuildTime = $currentTime
    Write-Host "`n[$([DateTime]::Now.ToString('HH:mm:ss'))] 检测到文件变化，开始构建..." -ForegroundColor Magenta
    
    # 执行构建命令
    try {
        Push-Location "$PSScriptRoot"
        npm run build
        Write-Host "`n[$([DateTime]::Now.ToString('HH:mm:ss'))] 构建完成!" -ForegroundColor Green
    } catch {
        Write-Host "`n[$([DateTime]::Now.ToString('HH:mm:ss'))] 构建失败: $($_.Exception.Message)" -ForegroundColor Red
    } finally {
        Pop-Location
    }
}

# 定义事件处理函数
$changedEventHandler = Register-ObjectEvent $watcher "Changed" -Action {
    $ext = [System.IO.Path]::GetExtension($Event.SourceEventArgs.FullPath)
    if ($allowedExtensions -contains $ext) {
        BuildProject
    }
}

$createdEventHandler = Register-ObjectEvent $watcher "Created" -Action {
    $ext = [System.IO.Path]::GetExtension($Event.SourceEventArgs.FullPath)
    if ($allowedExtensions -contains $ext) {
        BuildProject
    }
}

$deletedEventHandler = Register-ObjectEvent $watcher "Deleted" -Action {
    $ext = [System.IO.Path]::GetExtension($Event.SourceEventArgs.FullPath)
    if ($allowedExtensions -contains $ext) {
        BuildProject
    }
}

$renamedEventHandler = Register-ObjectEvent $watcher "Renamed" -Action {
    $oldExt = [System.IO.Path]::GetExtension($Event.SourceEventArgs.OldFullPath)
    $newExt = [System.IO.Path]::GetExtension($Event.SourceEventArgs.FullPath)
    if ($allowedExtensions -contains $oldExt -or $allowedExtensions -contains $newExt) {
        BuildProject
    }
}

# 初始构建一次
Write-Host "`n[$([DateTime]::Now.ToString('HH:mm:ss'))] 初始构建..." -ForegroundColor Magenta
BuildProject

# 保持脚本运行
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
} finally {
    # 清理事件处理器
    Unregister-Event -SourceIdentifier $changedEventHandler.Name
    Unregister-Event -SourceIdentifier $createdEventHandler.Name
    Unregister-Event -SourceIdentifier $deletedEventHandler.Name
    Unregister-Event -SourceIdentifier $renamedEventHandler.Name
    
    # 释放资源
    $watcher.Dispose()
    
    Write-Host "`n=== 自动构建脚本已停止 ===" -ForegroundColor Red
}
