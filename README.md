# Go

## 1. 宏观体系

### 1.1 学科定位

Go 后端开发的核心链路：

```text
HTTP 请求接入 → 业务处理 → 数据持久化
```

| 层级 | 技术 | 职责 |
| ------ |------ |------ |
| 接入层 | Gin | 接收请求、路由分发、参数解析、响应渲染 |
| 业务层 | Go 代码 | 业务逻辑、并发控制、错误处理 |
| 持久层 | GORM + MySQL | 数据存储、查询、事务 |

### 1.2 模块关系

```text
客户端请求
    │
    ▼
[Gin 接入层]    ← 路由、参数、中间件
    │
    ▼
[Go 业务层]     ← 逻辑、并发（errgroup）、日志（log）
    │
    ▼
[GORM 持久层]   ← 模型映射、CRUD
    │
    ▼
[MySQL 数据库]
```

一句话：Gin 负责“接客”，GORM 负责“存取”，MySQL 是“仓库”。

## 2. Gin 框架

### 2.1 引擎 Engine

`gin.Engine` 是 Gin 的核心对象，负责路由注册、中间件管理和请求分发。

```go
r := gin.Default()
r.GET("/ping", handler)
r.Run(":8080")
```

**边界：** 一个 `*gin.Engine` 实例通常只监听一个端口。要监听多个端口，需要创建多个独立引擎。

**坑：** 误以为同一个 `r` 可以多次调用 `r.Run()` 监听不同端口。`r.Run()` 是阻塞调用，第二个 `r.Run()` 通常不会被执行到。

```go
r := gin.Default()
r.Run(":8080") // 阻塞
r.Run(":8081") // 通常执行不到
```

### 2.2 gin.Default() vs gin.New()

| 对比项 | gin.Default() | gin.New() |
| ------ |------ |------ |
| Logger 中间件 | 自带 | 无 |
| Recovery 中间件 | 自带 | 无 |
| 适用场景 | 日常开发 | 需要精细控制中间件 |

**边界：** `gin.Default()` 自带的 `Recovery` 能捕获 handler 里的 panic 并返回 500，但不能阻止 `log.Fatal` 直接退出程序。

### 2.3 路由

```go
r.GET("/user", getUser)
r.POST("/user", createUser)
r.PUT("/user/:id", updateUser)
r.DELETE("/user/:id", deleteUser)
```

#### 路由组

```go
api := r.Group("/api/v1", AuthMiddleware())
{
    api.GET("/users", listUsers)
    api.GET("/articles", listArticles)
}
```

**坑：** 组前缀不要和内部路径重复，否则会出现 `/admin/admin` 这类双前缀路径。

#### 路由优先级

```text
静态路径 > 参数节点 :name > 通配节点 *path
```

```go
r.GET("/user/detail", h1)   // 优先匹配
r.GET("/user/:name", h2)    // 兜底
```

访问 `/user/detail` 走 `h1`；访问 `/user/张三` 走 `h2`。

**坑：** 重复注册会 panic。

```go
r.GET("/test", h1)
r.Any("/test", h2) // panic: handlers are already registered
```

兜底：

| 方法 | 触发时机 | 常用状态码 |
| ------ |------ |------ |
| `r.NoRoute` | 路径没匹配到 | 404 |
| `r.NoMethod` | 路径匹配但方法不对 | 405 |

### 2.4 参数处理

| 来源 | 位置 | 读取方法 |
| ------ |------ |------ |
| Query | URL `?key=value` | `c.Query("key")` |
| Form | 请求体 | `c.PostForm("key")` |
| Path | URL 路径段 | `c.Param("key")` |

带默认值版本：`c.DefaultQuery`、`c.DefaultPostForm`。

#### ShouldBind

```go
type LoginForm struct {
    Username string `form:"username" binding:"required,min=3,max=20"`
    Email    string `form:"email" binding:"required,email"`
}

if err := c.ShouldBind(&form); err != nil {
    c.JSON(400, gin.H{"err": err.Error()})
    return
}
```

**坑：**

- 结构体字段首字母必须大写。
- 必须传指针 `&form`。
- 常用 tag：`required`、`email`、`min`、`max`、`gt`、`lt`、`oneof`。

### 2.5 中间件

```go
func LogMiddleware(c *gin.Context) {
    fmt.Println("请求开始:", c.Request.URL.Path)
    c.Next()
    fmt.Println("请求结束:", c.Request.URL.Path)
}
```

| 方法 | 作用 |
| ------ |------ |
| `c.Next()` | 继续执行后续中间件和处理函数 |
| `c.Abort()` | 中断后续执行 |

**坑：** 只写 `c.Abort()` 不写 `return`，后面的代码仍会执行。

**黄金组合：**

```go
c.JSON(401, ...)
c.Abort()
return
```

#### 中间件中使用 goroutine

**坑：** 不能在 goroutine 里直接使用 `c`，因为 Gin 会复用 `gin.Context`。

安全写法：

```go
path := c.Request.URL.Path
go func(p string) {
    // 使用 p
}(path)
```

或者：

```go
cCp := c.Copy()
go func() {
    fmt.Println(cCp.Request.URL.Path)
}()
```

**边界：** `c.Copy()` 返回的是副本，不能用来写响应，也不能调用 `Next()` / `Abort()`。

### 2.6 响应渲染

```go
c.JSON(200, gin.H{"code": 0, "msg": "ok"})
```

| 方法 | 特点 |
| ------ |------ |
| `c.JSON` | HTML 特殊字符会转义 |
| `c.PureJSON` | 不转义，返回原始字符 |

**坑：** `c.JSON` 之后通常要 `return`，否则后续代码可能触发重复写响应头。

### 2.7 文件上传

```go
r.POST("/upload", func(c *gin.Context) {
    file, err := c.FormFile("avatar")
    if err != nil {
        c.JSON(400, gin.H{"err": err.Error()})
        return
    }
    dst := filepath.Join("./uploads", filepath.Base(file.Filename))
    c.SaveUploadedFile(file, dst)
})
```

**坑：**

- 前端 `<form>` 必须写 `enctype="multipart/form-data"`。
- 前端 `name` 和后端 `c.FormFile` 参数必须一致。
- 文件名建议用 `filepath.Base` 防止路径穿越。

## 3. 并发服务管理：errgroup

### 3.1 为什么需要 errgroup

`r.Run()` 是阻塞调用，第二个服务无法直接启动：

```go
r1.Run(":8080") // 阻塞
r2.Run(":8081") // 通常执行不到
```

朴素方案：

```go
go r1.Run(":8080")
go r2.Run(":8081")
select {}
```

问题：看不到服务启动失败的错误。

### 3.2 推荐写法

```go
var g errgroup.Group

g.Go(func() error {
    return r1.Run(":8080")
})

g.Go(func() error {
    return r2.Run(":8081")
})

if err := g.Wait(); err != nil {
    log.Fatal(err)
}
```

### 3.3 errgroup vs WaitGroup

| 能力 | sync.WaitGroup | errgroup.Group |
| ------ |------ |------ |
| 等待结束 |  |  |
| 收集错误 |  | 返回第一个错误 |
| Context 取消 |  | 可用 `WithContext` |

**坑：** 必须用 `g.Wait()` 阻塞主 goroutine，否则主程序退出后所有服务会被强制终止。

### 3.4 两个服务为什么能返回不同内容

根本原因：**两个独立 engine + 两个不同 handler**。

- `r1` 和 `r2` 是两个独立实例，各自有自己的路由表。
- 请求 `8080` 走 `r1` 的 handler。
- 请求 `8081` 走 `r2` 的 handler。

**反例：** 同一个 `r` 不能同时监听两个端口，也不能靠同一个 engine 返回两套完全不同的根路由响应。

## 4. 数据库基础：SQL vs MySQL

| 概念 | 定义 | 类比 |
| ------ |------ |------ |
| SQL | 查询语言 | 普通话 |
| MySQL | 数据库软件 | 使用普通话的人 |

MySQL、PostgreSQL、SQLite、Oracle 等都支持 SQL。

**学习顺序建议：**

1. 先学 SQL 基础：`SELECT`、`INSERT`、`UPDATE`、`DELETE`。
2. 再装 MySQL，本地运行一个数据库实例。
3. 最后接 GORM，用 Go 代码操作 MySQL。

**坑：** 不要死磕完全部 SQL 理论再学 GORM。GORM 的每条操作背后都对应 SQL，遇到不懂的再查效率更高。

## 5. GORM

### 5.1 连接数据库

DSN 示例：

```text
root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local
```

| 参数 | 作用 | 漏了会怎样 |
| ------ |------ |------ |
| `charset=utf8mb4` | 支持中文和 emoji | 中文可能乱码 |
| `parseTime=True` | 时间转 `time.Time` | 时间字段可能变成 `[]byte` |
| `loc=Local` | 使用本地时区 | 时区可能错误 |

```go
dsn := "root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local"
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
if err != nil {
    log.Fatal("连接失败: ", err)
}
```

**坑：**

- `Access denied`：用户名或密码错误。
- `Unknown database`：数据库不存在。
- `connection refused`：MySQL 未启动。

### 5.2 模型定义

```go
type User struct {
    gorm.Model
    Name string
}
```

`gorm.Model` 包含：

| 字段 | 作用 |
| ------ |------ |
| `ID` | 主键，自增 |
| `CreatedAt` | 创建时间 |
| `UpdatedAt` | 更新时间 |
| `DeletedAt` | 软删除 |

常用 tag：

```go
gorm:"primaryKey"
gorm:"type:varchar(50)"
gorm:"not null"
gorm:"uniqueIndex"
gorm:"default:0"
```

**坑：**

- tag 必须用反引号 ```。
- `primaryKey` 的 `K` 必须大写。
- `uint` 不要拼成 `unit`。

### 5.3 自动建表

```go
db.AutoMigrate(&User{})
```

| 行为 | 说明 |
| ------ |------ |
| 幂等 | 反复调用不会重复建表 |
| 只增不删 | 新增字段会加列，删除字段不会删列 |
| 不改类型 | 字段类型变化不会自动修改 |

**坑：** 必须传指针 `&User{}`，不能传值 `User{}`。

## 6. 类型选型：int、uint

| 类型 | 取值范围 | 说明 |
| ------ |------ |------ |
| `int` | 可正、可负、可为 0 | 日常首选 |
| `uint` | ≥ 0 | 明确无负数时使用 |
| `int64` | 固定 64 位有符号 | 跨平台更稳定 |
| `uint64` | 固定 64 位无符号 | 大 ID、计数 |

**坑：** `int` 和 `uint` 不能直接混用。

```go
var a int = 10
var b uint = 20
c := uint(a) + b // 必须显式转换
```

**边界：** `uint(-1)` 不会得到 `-1`，而会变成很大的无符号值。

GORM 主键常用 `uint`，因为主键通常是自增正整数；但如果需要与其他系统交互，主键用 `int64` 更稳妥。

## 7. 日志与程序退出

| 函数 | 行为 | 是否退出 | 是否执行 defer | 能否 recover |
| ------ |------ |------ |------ |------ |
| `log.Print` | 打印日志 | 否 | 是 | — |
| `log.Panic` | 打印 + panic | 否 | 是 | 能 |
| `log.Fatal` | 打印 + `os.Exit(1)` | 是 | 否 | 否 |

```go
if err := g.Wait(); err != nil {
    log.Fatal(err)
}
```

**坑：**

- `log.Fatal` 不会执行 `defer`。
- `log.Fatal` 不能被 `recover` 捕获。
- Gin 的 `Recovery` 中间件能捕获 panic，但不能阻止 `log.Fatal` 退出。

**适用边界：** `log.Fatal` 适合启动阶段失败；不适合业务 handler 里的普通错误。

## 8. 常见坑汇总

| 场景 | 坑 | 建议 |
| ------ |------ |------ |
| Gin 多端口 | 同一个 `r` 调两次 `r.Run()` | 用多个 `gin.Engine` |
| errgroup | 忘记 `g.Wait()` | 主 goroutine 会被提前退出 |
| ShouldBind | 传值不传指针 | 必须写 `&form` |
| 中间件 goroutine | 直接用 `c` | 用 `c.Copy()` 或提前取值 |
| GORM DSN | 漏写 `parseTime=True` | 时间字段读写异常 |
| AutoMigrate | 忘记 `&` | 必须传结构体指针 |
| log.Fatal | 用在业务错误里 | 会导致整个服务退出 |
| int/uint | 直接混用 | 需要显式转换 |

## 9. 后续可扩展方向

- 补充 GORM 连接 MySQL 的完整 CRUD 示例。
- 补充 errgroup 启动多 Gin 服务的可运行示例。
- 补充统一响应结构、错误码设计。
- 补充 JWT 鉴权、分页查询、日志中间件。
# Go 模板引擎完全教程

> 本文档系统整理 Go 语言 `html/template` 模板引擎的核心知识，涵盖：解决什么问题、核心机制、语法体系、复用方式、常见坑、边界情况、设计哲学与深度理解。

## 一、解决什么问题？

### 1.1 核心问题：动态生成 HTML

**没有模板引擎时：**
```go
html := "<h1>欢迎你，" + name + "</h1>"
w.Write([]byte(html))
- 字符串拼接，页面一复杂就乱。
- 前后端代码混在一起。
- 用户输入 `<script>` 会直接执行（XSS 攻击）。
``` 
**有了模板引擎：**
```
html
<h1>欢迎你，{{ .Name }}</h1>
页面结构和数据分离。
```

- 页面结构和数据分离。
- 自动转义危险字符。
- 模板可复用、可继承。

### 1.2 一句话总结
模板引擎 = 把“页面长什么样”和“数据是什么”分开，用占位符在中间搭桥。

---

## 二、核心逻辑与三步走

### 2.1 三步走
```text
定义模板 → 解析模板 → 渲染模板
（写文件） （加载进内存）（填数据、输出）
```

### 2.2 数据流向
```text
Go 数据（结构体/map）
        │
        ▼
   模板文件（含 {{ }} 占位符）
        │
        ▼
    渲染（用数据替换占位符）
        │
        ▼
  最终 HTML → 浏览器
```

### 2.3 核心概念对应

| 概念 | 作用 | 代码 |
|---|---|---|
| 模板文件 | 页面骨架 + 占位符 | `index.tmpl` |
| `.` | 当前数据对象 | `{{ .Name }}` |
| `ParseFiles` / `ParseGlob` | 加载模板 | `template.ParseFiles(...)` |
| `Execute` / `ExecuteTemplate` | 渲染并输出 | `tmpl.Execute(w, data)` |
| `define` | 定义可复用块 | `{{ define "content" }}...{{ end }}` |
| `block` | 定义块 + 引入块 | `{{ block "content" . }}{{ end }}` |
| `template` | 引入其他模板 | `{{ template "base.tmpl" }}` |

### 2.4 最小可用流程
```go
// 1. 解析模板
tmpl, err := template.ParseFiles("./index.tmpl")
if err != nil { return }

// 2. 准备数据
data := map[string]interface{}{"Name": "张三"}

// 3. 渲染到响应
tmpl.Execute(w, data)
```

### 2.5 带继承的流程
```text
base.tmpl（骨架，含 block "content"）
    ▲
    │ template 引用
    │
index.tmpl（定义 content）

执行：tmpl.ExecuteTemplate(w, "index.tmpl", data)
```

---

## 三、语法体系

### 3.1 语法速查

| 语法 | 作用 |
|---|---|
| `{{ . }}` | 打印当前数据 |
| `{{ .Field }}` | 访问字段 |
| `{{ $x := .Name }}` | 声明变量 |
| `{{ if pipeline }}...{{ end }}` | 条件 |
| `{{ range pipeline }}...{{ end }}` | 循环 |
| `{{ with pipeline }}...{{ end }}` | 切换上下文 |
| `{{/* 注释 */}}` | 注释 |
| `{{- ... -}}` | 去除空白 |
| `{{ printf "%s" .Name }}` | 函数调用 |
| `{{ .Name \| safe }}` | 管道 |

### 3.2 三种“切换上下文”的对比

| 语法 | 是否切换 `.` | 是否输出 | 典型场景 |
|---|---|---|---|
| `if` | ❌ 不切换 | 条件输出 | 判断是否显示 |
| `range` | ✅ 切换成当前元素 | 循环输出 | 遍历列表 |
| `with` | ✅ 切换成指定值 | 条件输出 | 简化深层字段访问 |

**记忆口诀：**
`if` 只管判断，`range` 遍历切换，`with` 简化访问。

### 3.3 变量与管道
```text
{{ $title := .Title }}          <!-- 声明变量 -->
{{ .Name | printf "%s真帅" }}    <!-- 管道：前一个结果传给后一个函数 -->
{{- .Name -}}                    <!-- 去除左右空白 -->
```

### 3.4 条件判断
```html
{{ if .Age }}
    
{{ else }}
    
{{ end }}
```
**零值判断规则：** `0`、`""`、`nil`、空切片/空 map 都算假。

### 3.5 遍历
```html
{{ range .Users }}
    
{{ else }}
    
{{ end }}
```
`range` 内部 `.` 是当前元素；外层数据用 `$` 访问。

### 3.6 with
```html
{{ with .User }}
    
    
{{ end }}
```
`with` 把 `.` 切换成 `.User`，内部直接写 `.Name`。

---

## 四、复用机制

### 4.1 两种复用层次

| 层次 | 语法 | 场景 |
|---|---|---|
| 片段复用 | `{{ template "xxx" }}` | 引入一个组件（导航、页脚） |
| 继承复用 | `{{ block "xxx" . }}` | 子模板覆盖父模板的某个区域 |

### 4.2 define + template

**同文件内：**
```html
{{ define "ol.tmpl" }}

{{ end }}

{{ template "ol.tmpl" }}
```

**外部文件：**
```html
<!-- ul.tmpl -->


<!-- t.tmpl -->
{{ template "ul.tmpl" }}
```

**解析时：**
```go
tmpl, err := template.ParseFiles("./t.tmpl", "./ul.tmpl")
```
> **注意：** 被引用的模板必须和主模板一起被解析。

### 4.3 block 模板继承

**base.tmpl：**
```html
<!DOCTYPE html>

```

**index.tmpl：**
```html
{{ template "base.tmpl" }}

{{ define "content" }}
    
{{ end }}
```

**渲染：**
```go
tmpl.ExecuteTemplate(w, "index.tmpl", nil)
```

---

## 五、自定义函数与安全

### 5.1 自定义函数

**定义：**
```go
kua := func(arg string) (string, error) {
    return arg + "真帅", nil
}
```

**注册：**
```go
tmpl, err := template.New("hello").
    Funcs(template.FuncMap{"kua": kua}).
    ParseFiles("./hello.html")
```

**使用：**
```html
{{ kua .Name }}
```

**规则：**
- 函数必须返回 1 个或 2 个值。
- 如果返回 2 个，第二个必须是 `error`。
- 注册必须在 `Parse` 之前。

### 5.2 html/template 的安全机制

`html/template` 会自动转义危险字符，防止 XSS 攻击。

**转义示例：**
- 传入 `<script>alert('xss')</script>`
- 渲染后变成 `&lt;script&gt;alert('xss')&lt;/script&gt;`

**关闭转义（慎用）：**
```go
tmpl, _ := template.New("xss.tmpl").Funcs(template.FuncMap{
    "safe": func(s string) template.HTML {
        return template.HTML(s)
    },
}).ParseFiles("./xss.tmpl")
```
```html
{{ . | safe }}
```
> **警告：** 只有完全信任内容来源时才能用 `safe`。

---

## 六、常见坑与边界

### 6.1 常见坑

| 坑 | 现象 | 原因 | 解决 |
|---|---|---|---|
| `define` 命名冲突 | 访问 `/about` 却显示 `/` 的内容 | 多个文件定义同名 `define`，后加载的覆盖先加载的 | 每个 handler 用 `ParseFiles` 只加载自己需要的文件 |
| 引用名 ≠ 文件名 | `no such template "base.html"` | 文件叫 `base.tmpl`，引用写成了 `"base.html"` | 引用名 = 文件名（含后缀） |
| `block` 少传参数 | `wrong number of args for block` | 写成 `{{ block "content" }}`，漏了 `.` | `{{ block "content" . }}` |
| 字段小写访问不到 | `can't evaluate field name` | 结构体字段小写 | 字段首字母必须大写 |
| 出错没 return | 解析失败后 panic | 出错后没有 return，继续执行 | 错误检查后立刻 return |
| `Funcs` 放错位置 | 模板里函数未定义 | `Funcs` 放在了 `Parse` 后面 | `New().Funcs().Parse()` |

### 6.2 边界情况

| 边界 | 行为 |
|---|---|
| 数据是 `nil` | 输出 `<no value>` |
| 字段不存在 | 运行时报错 |
| 空切片 `range` | 什么都不输出（除非有 `else`） |
| 空切片 `range` + `else` | 执行 `else` 分支 |
| HTML 标签 | `html/template` 自动转义 |
| `{{ . }}` 打印结构体 | 输出 `{字段1 字段2}` 格式 |
| `{{ . }}` 打印 map | 输出 `map[key:value]` 格式 |

---

## 七、知识框架全景图

```text
┌─────────────────────────────────────────────────────────┐
│                    Go 模板引擎                            │
│                                                         │
│  ┌─────────────┐                                        │
│  │  为什么用    │  动态 HTML / 前后端分离 / 防 XSS        │
│  └──────┬──────┘                                        │
│         │                                               │
│  ┌──────▼──────┐                                        │
│  │  核心机制    │  定义 → 解析 → 渲染                     │
│  │             │  数据流：Go数据 → 模板 → HTML            │
│  └──────┬──────┘                                        │
│         │                                               │
│  ┌──────▼──────┐                                        │
│  │  语法体系    │                                        │
│  │             │                                        │
│  │  ┌────────┐ │  取值：{{.}}、{{.Field}}                │
│  │  │ 基础    │ │  变量：{{$x := ...}}                   │
│  │  └────────┘ │                                        │
│  │  ┌────────┐ │  条件：if                              │
│  │  │ 逻辑    │ │  循环：range                           │
│  │  └────────┘ │  上下文：with                          │
│  │  ┌────────┐ │  片段：template                        │
│  │  │ 复用    │ │  继承：block                           │
│  │  └────────┘ │  自定义：Funcs                          │
│  │  ┌────────┐ │  管道：|                               │
│  │  │ 进阶    │ │  空白：{{- -}}                         │
│  │  └────────┘ │  安全：自动转义 / safe                  │
│  └──────┬──────┘                                        │
│         │                                               │
│  ┌──────▼──────┐                                        │
│  │  约束与坑    │  全局命名空间 / 大小写 / 加载顺序        │
│  └──────┬──────┘                                        │
│         │                                               │
│  ┌──────▼──────┐                                        │
│  │  边界情况    │  nil / 空值 / 不存在字段 / 转义          │
│  └─────────────┘                                        │
└─────────────────────────────────────────────────────────┘
```

---

## 八、深度理解：三个“为什么”

### 8.1 为什么 html/template 要自动转义？
因为 Web 的第一安全威胁是 XSS（跨站脚本攻击）。

如果不转义，用户在评论区输入：
```html

所有访问这个页面的用户，cookie 都会被偷走。

`html/template` 自动把 `<` 转成 `&lt;`，脚本就变成普通文字，无法执行。

**安全是默认行为**，关闭它需要显式声明（`template.HTML`）。这是“安全优先”的设计哲学。

### 8.2 为什么 define 是全局的？
因为模板引擎的设计目标是“简单”。

如果 `define` 是局部的，就需要引入作用域、命名空间、导入系统……复杂度飙升。

Go 选择了“全局 + 覆盖”的简单模型。代价是：多个文件定义同名块会冲突。但这不是 bug，是设计取舍。

**解决办法：** 每个 handler 只加载自己需要的模板。

### 8.3 为什么 range 内部的 . 会变？
因为 `range` 的本质是“迭代”。

```text
{{ range .Users }}
    {{ . }}      ← 这里应该是“当前用户”，而不是“整个数据”
{{ end }}
```
如果 `.` 不变，你就没法直接访问当前元素，每次都要写 `{{ index $.Users $i }}`，非常难用。

所以 Go 设计成：`range` 内部 `.` 切换成当前元素，外层数据用 `$` 访问。

---

## 九、总结

Go 模板引擎的本质是“填空器”，设计哲学是“数据驱动 + 关注点分离”。核心机制是“定义→解析→渲染”三步。语法体系分四层：取值、逻辑、复用、进阶。所有约束和坑都源于“全局命名空间 + 运行时检查 + 安全优先”三大设计取舍。
