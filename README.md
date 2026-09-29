# doccheck — Markdown 文档引用审计工具

`doccheck` 扫描本地 Markdown 文档树，检查失效的文件引用、章节锚点，
以及从指定入口页面无法到达的页面。它只读取文档并输出 JSON 报告，
**从不修改被检查的文件**，也不访问网络、不跟随符号链接。

## 构建

```
go build ./cmd/doccheck
```

## 用法

```
doccheck -root <文档根目录> -entry <入口 .md（相对 root 的路径）> -report <JSON 报告路径>
```

示例：

```
doccheck -root examples/docs -entry index.md -report out/report.json
```

### 退出码

| 码 | 含义 |
|----|------|
| 0 | 无失效链接、无不可达页面；不生成报告文件 |
| 1 | 存在失效链接或不可达页面；报告已写入 `-report` |
| 2 | 输入错误（参数缺失、根目录/入口非法、报告路径在根目录内等）；不生成报告 |

安全约束：

- 只扫描根目录内的普通 `.md` 文件；符号链接文件与符号链接目录一律跳过。
- 报告路径必须位于根目录之外，避免覆盖被检查文档。

## 支持的 Markdown 子集

- **ATX 标题**：独占一行、由 1–6 个 `#` 紧跟一个空格开头，正文为纯文本。
- **简单链接**：`[文字](目标)`；目标不含空白或圆括号。图片 `![]()`、
  引用式链接和 HTML 均不处理。
- **围栏代码块**：去除首尾空白后恰为 ` ``` ` 或 `~~~` 的行开启，
  闭合符必须与开启符一致；块内的标题与链接全部忽略。
- **行内代码**：成对单反引号 `` `...` `` 内的内容忽略；未配对的反引号
  视为普通字符。

## 链接与锚点规则

- `http://`、`https://`、`mailto:` 链接直接忽略。
- 其他目标由相对路径加可选的 `#锚点` 组成；路径相对所在文件解析，
  空路径表示本文件。不做百分号解码。
- 拒绝绝对路径、反斜杠、查询串（`?`）、盘符/冒号；`.`/`..` 可规范化，
  但规范化后越出根目录即判为越界。目标路径任一级为符号链接判为非法目标。
- 本地目标必须是存在的普通文件；带锚点时目标必须是 Markdown 文件且锚点存在。

### 锚点生成

转小写 → 保留 Unicode 字母与数字 → 空白转 `-` → 删除其他字符
→ 合并并去除首尾连字符；空结果使用 `section`。
同文档内锚点按出现顺序唯一，冲突时追加 `-2`、`-3`……例如：

```markdown
## Setup
## Setup
```

生成 `setup` 与 `setup-2`。链接中的锚点按原文精确匹配，不再转小写。

## 可达性

有效的 Markdown 目标构成有向页面图，从入口页沿有效链接进行 BFS。
未被访问到的 `.md` 文件列入报告的 `unreachable`（按相对路径排序）。
自链接不增加可达页面。

## 报告格式

```json
{
  "root": "examples/docs",
  "entry": "index.md",
  "links": [
    {
      "source": "index.md",
      "line": 3,
      "order": 1,
      "target": "getting-started.md#setup",
      "path": "getting-started.md",
      "anchor": "setup",
      "status": "ok"
    }
  ],
  "unreachable": ["orphan.md"],
  "counts": {
    "total_links": 6,
    "ok": 4,
    "outside_root": 0,
    "file_missing": 1,
    "illegal_target": 0,
    "anchor_missing": 0,
    "unreachable": 1
  }
}
```

链接按源文件相对路径、行号、同行出现顺序排列。`status` 取值：

- `ok`：有效
- `outside_root`：路径规范化后越出根目录
- `file_missing`：目标文件不存在或不是普通文件
- `illegal_target`：语法非法或路径中包含符号链接
- `anchor_missing`：文件存在但锚点不存在（或非 Markdown 文件带锚点）

## 测试

```
go test ./...
```

测试覆盖重名标题锚点、代码块/行内代码排除、跨页锚点、循环引用、
越界路径、不可达页面与符号链接目标。
