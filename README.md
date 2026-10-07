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
