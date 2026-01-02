# Logger Package

统一的日志模块，支持多级别日志输出和环境配置。

## 功能特性

- ✅ 多级别日志：DEBUG、INFO、WARN、ERROR、FATAL
- ✅ 分离输出：INFO/DEBUG → stdout，WARN/ERROR/FATAL → stderr
- ✅ 环境配置：根据 ENV 和 DEBUG 环境变量自动调整日志级别
- ✅ 时间戳和文件位置：自动记录日志时间和代码位置
- ✅ 兼容标准库：提供 Println、Printf 等兼容函数

## 快速开始

### 基础使用

```go
package main

import "github.com/chengchuu/go-gin-gee/pkg/logger"

func main() {
    // 初始化（自动根据环境变量配置）
    logger.Init()
    
    // 使用不同级别的日志
    logger.Debug("Debug message:  %v", someData)
    logger.Info("Server started on port %d", 3000)
    logger.Warn("High memory usage:  %d%%", 85)
    logger.Error("database connection failed:  %v", err)
    
    // 致命错误（会退出程序）
    if criticalError != nil {
        logger.Fatal("Critical error: %v", criticalError)
    }
}
```

### 在现有代码中替换

**替换前**：
```go
import "log"

log.Println("Server starting...")
log.Println("Error:", err)
```

**替换后**：
```go
import "github.com/chengchuu/go-gin-gee/pkg/logger"

logger.Info("Server starting...")
logger.Error("Error:  %v", err)
```

## 日志级别

| 级别 | 用途 | 输出流 | 示例 |
|------|------|--------|------|
| DEBUG | 调试信息 | stdout | `logger.Debug("Variable x = %d", x)` |
| INFO | 一般信息 | stdout | `logger.Info("Server started")` |
| WARN | 警告信息 | stderr | `logger.Warn("High memory:  %d%%", 85)` |
| ERROR | 错误信息 | stderr | `logger.Error("DB failed: %v", err)` |
| FATAL | 致命错误 | stderr + exit | `logger.Fatal("Cannot start")` |

## 环境配置

### 自动配置（推荐）

```go
// 初始化时自动根据环境变量配置
logger.Init()

// 开发环境：ENV=development → DEBUG 级别
// 生产环境：ENV=production → INFO 级别
// 强制 DEBUG：DEBUG=true → DEBUG 级别
```

### 手动配置

```go
// 设置日志级别
logger.SetLevel(logger.DEBUG)

// 获取当前级别
level := logger.GetLevel()
```

### Docker/Supervisor 配置

```ini
# supervisor 配置
[program:goapi]
command=/web/api
environment=ENV="production",DEBUG="false"

# DEBUG 模式
environment=ENV="development",DEBUG="true"
```

## 输出示例

### 开发环境（DEBUG 级别）

```
[DEBUG] 2025/12/30 15:30:01 main.go:15: Loading configuration from data/config.json
[INFO]  2025/12/30 15:30:01 main.go:20: Server starting on port 3000
[INFO]  2025/12/30 15:30:02 api.go:45: Database connected
[WARN]  2025/12/30 15:30:10 handler.go:78:  Slow query detected:  500ms
[ERROR] 2025/12/30 15:30:15 service.go:32: Redis connection timeout
```

### 生产环境（INFO 级别）

```
[INFO]  2025/12/30 15:30:01 main.go:20: Server starting on port 3000
[INFO]  2025/12/30 15:30:02 api.go:45: Database connected
[WARN]  2025/12/30 15:30:10 handler.go:78: Slow query detected: 500ms
[ERROR] 2025/12/30 15:30:15 service.go:32: Redis connection timeout
```

## 与 Supervisor 集成

```ini
[program:goapi]
command=/web/api --config-path="/web/data/config.prd.json"
directory=/web

# INFO 和 DEBUG 日志
stdout_logfile=/web/log/supervisor/api.log

# WARN、ERROR、FATAL 日志
stderr_logfile=/web/log/supervisor/api_error.log
```

## 迁移指南

### 1. 在 main.go 初始化

```go
package main

import (
    "github.com/chengchuu/go-gin-gee/pkg/logger"
    // ... 其他导入
)

func main() {
    // ✅ 在程序启动时初始化
    logger.Init()
    
    logger.Info("Application starting...")
    
    // ...  原有代码
}
```

### 2. 替换现有 log 调用

```bash
# 查找项目中所有使用 log 的地方
grep -r "log\." --include="*.go" .

# 批量替换（需要手动确认）
# log.Println → logger.Info
# log.Printf → logger.Info
# log.Fatal → logger.Fatal
```

### 3. 分类日志级别

```go
// ❌ 替换前（全部用 log）
log.Println("Server starting")
log.Println("Warning:  high memory")
log.Println("Error:", err)

// ✅ 替换后（按级别分类）
logger.Info("Server starting")
logger.Warn("High memory usage: %d%%", memPercent)
logger.Error("Operation failed: %v", err)
```

## API 参考

### 初始化函数

- `Init()` - 初始化全局 logger（根据环境变量）
- `New(stdout, stderr, level)` - 创建自定义 logger

### 日志函数

- `Debug(format, v...)` - 调试日志
- `Info(format, v...)` - 信息日志
- `Warn(format, v...)` - 警告日志
- `Error(format, v...)` - 错误日志
- `Fatal(format, v...)` - 致命错误（会退出程序）
- `Println(v...)` - 兼容标准库（输出到 INFO）
- `Printf(format, v...)` - 兼容标准库（输出到 INFO）

### 配置函数

- `SetLevel(level)` - 设置日志级别
- `GetLevel()` - 获取当前日志级别
- `LevelFromString(level)` - 从字符串解析日志级别

## 测试

```bash
# 运行测试
go test ./pkg/logger/

# 查看覆盖率
go test -cover ./pkg/logger/

# 详细测试输出
go test -v ./pkg/logger/
```

## 最佳实践

1. ✅ 在 `main.go` 启动时调用 `logger.Init()`
2. ✅ 根据语义选择日志级别
3. ✅ 错误日志使用 `%v` 格式化 error
4. ✅ 生产环境设置 `ENV=production`
5. ✅ 调试时设置 `DEBUG=true`
6. ✅ 使用 Supervisor 分离 stdout 和 stderr 日志

## 示例项目

查看完整示例：
- [internal/api/api.go](../../internal/api/api.go) - 主入口初始化
- [internal/api/controllers/](../../internal/api/controllers/) - 控制器中使用
- [internal/pkg/config/](../../internal/pkg/config/) - 配置模块中使用
