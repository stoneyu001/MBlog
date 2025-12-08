# ============================================
# MBlog 快速启动脚本 (Windows PowerShell)
# ============================================

Write-Host "🚀 MBlog 快速启动脚本" -ForegroundColor Cyan
Write-Host "====================" -ForegroundColor Cyan
Write-Host ""

# 检查 Docker 是否安装
try {
    docker --version | Out-Null
    Write-Host "✅ Docker 环境检测通过" -ForegroundColor Green
} catch {
    Write-Host "❌ 错误：未检测到 Docker，请先安装 Docker Desktop" -ForegroundColor Red
    Write-Host "   下载地址：https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# 检查 .env 文件是否存在
if (-Not (Test-Path .env)) {
    Write-Host "⚠️  未找到 .env 文件，正在从模板创建..." -ForegroundColor Yellow
    if (Test-Path .env.example) {
        Copy-Item .env.example .env
        Write-Host "✅ 已创建 .env 文件" -ForegroundColor Green
        Write-Host "⚠️  请编辑 .env 文件，设置数据库密码！" -ForegroundColor Yellow
        Write-Host "   位置: $(Get-Location)\.env" -ForegroundColor Yellow
        Write-Host ""
        Read-Host "按回车键继续"
    } else {
        Write-Host "❌ 错误：未找到 .env.example 模板文件" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✅ 找到 .env 配置文件" -ForegroundColor Green
}

Write-Host ""
Write-Host "📦 正在启动服务..." -ForegroundColor Cyan
Write-Host "   1️⃣  PostgreSQL 数据库"
Write-Host "   2️⃣  Go 后端服务"
Write-Host "   3️⃣  Nginx 前端服务"
Write-Host ""

# 启动服务
docker-compose up -d

# 检查启动状态
if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ 服务启动成功！" -ForegroundColor Green
    Write-Host ""
    Write-Host "📌 访问地址：" -ForegroundColor Cyan
    
    # 读取端口配置
    $envContent = Get-Content .env
    $frontendPort = ($envContent | Select-String "FRONTEND_PORT=").ToString().Split("=")[1]
    $backendPort = ($envContent | Select-String "BACKEND_PORT=").ToString().Split("=")[1]
    
    Write-Host "   • 前端网站: http://localhost:$frontendPort" -ForegroundColor White
    Write-Host "   • 后端 API: http://localhost:$backendPort" -ForegroundColor White
    Write-Host "   • 管理界面: http://localhost:$backendPort/admin" -ForegroundColor White
    Write-Host ""
    Write-Host "📊 查看服务状态: docker-compose ps" -ForegroundColor Yellow
    Write-Host "📋 查看服务日志: docker-compose logs -f" -ForegroundColor Yellow
    Write-Host "🛑 停止服务: docker-compose down" -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "❌ 服务启动失败，请查看错误信息" -ForegroundColor Red
    Write-Host "💡 提示：" -ForegroundColor Yellow
    Write-Host "   • 检查端口是否被占用"
    Write-Host "   • 查看日志: docker-compose logs"
    exit 1
}
