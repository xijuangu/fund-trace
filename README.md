# fund-trace

中国公募基金 & A 股实时跟踪终端工具。纯 Go 单二进制，SQLite 持久化，Bubble Tea TUI 交互仪表盘。下载即用，零外部依赖。

## 特性

**双资产支持**
- 基金实时估值（天天基金）+ 历史净值（东方财富）
- A 股实时行情（腾讯财经非正式接口），沪深市场自动识别
- 统一 TUI 仪表盘混合展示基金与股票

**TUI 交互操作**
- 光标导航（j/k / ↑↓），选中行反显高亮
- `a` 添加基金：输入 6 位代码，自动发现基金名称，立即生效并写入配置
- `a` 添加股票：输入 `sh600519` 或 `sz000001` 格式，或 `stock:sh:600519`
- `d` 删除资产：二次确认后移除，同步更新配置文件
- `A` 设置涨跌告警：切换跌/涨类型，输入阈值百分比
- `s` 打开设置面板：实时编辑刷新间隔、并发数等参数，Esc 保存到 config.yaml
- `Enter` 查看基金详情：全屏展示 SMA/RSI 趋势分析 + 最近 10 天历史净值
- `h` 快捷键帮助浮层，随时查看所有可用操作

**数据获取**
- 并发拉取天天基金实时估值，所有基金同时请求，1-2 秒内返回
- 自动发现基金名称——只需 6 位代码，系统从东方财富 26400+ 基金数据库中匹配全称
- 腾讯财经批量股票行情，股票列表一次请求
- 支持历史净值查询，默认 30 天，可自定义天数

**Sparkline 趋势线**
- 0% 锚定绝对坐标系：▁▂▃▄▅▆▇█，0% 固定在 ▄，正数往上、负数往下
- 数据源为日涨跌幅，每个块直接反映当天涨跌幅度
- 逐块独立染色：正值绿色、负值红色、▄（0% 附近）灰色
- 最右块自动追加当日实时估值，始终与 Change% 列同步

**持久化**
- SQLite WAL 模式存储，崩溃安全
- 五张表：`funds`、`assets`、`nav_snapshots`、`alerts`、`daily_summary`
- 首次运行自动建库建表
- TUI 增删操作自动写回 `config.yaml`，重启不丢失

**分析**
- SMA(5/10/20) 简单移动平均 + EMA 指数移动平均
- RSI(14) 相对强弱指数，含超买 (>70) / 超卖 (<30) 标识
- 趋势方向判定：SMA5 vs SMA20 交叉

**告警**
- 跌幅 / 涨幅告警，桌面通知（macOS 通知中心）
- 冷却机制，同一告警在冷却期内不重复推送

**导出**
- CSV / HTML 双格式，含 Type/Market/Code/Name/PriceOrNAV/Previous/Change%/UpdateTime
- 文件名自动带日期戳

## 快速开始

```bash
go build -o fund-trace .
./fund-trace
```

首次运行时若没有 `config.yaml`，会自动生成一份包含 13 只默认基金的配置文件。`fund-trace.db` 也会同时创建在当前目录。

## CLI 命令参考

### `fund-trace`（默认）

启动全屏 TUI 交互仪表盘。

### `fund-trace list`

表格形式列出所有基金和股票的当前实时数据。

```bash
$ fund-trace list
```

### `fund-trace add <code>`

添加基金，自动发现名称。

```bash
$ fund-trace add 000001
Added fund 000001: 华夏成长混合
```

### `fund-trace stock add <code>`

添加 A 股股票。市场自动推断：6 开头 → sh，0/3 开头 → sz。也可显式指定。

```bash
$ fund-trace stock add 600519        # 自动推断 sh
$ fund-trace stock add sh 600519     # 显式指定
$ fund-trace stock add 000001        # 自动推断 sz
$ fund-trace stock add 300750        # 自动推断 sz
```

### `fund-trace remove <code>`

移除基金。别名：`rm`。

```bash
$ fund-trace remove 000001
Removed fund 000001
```

### `fund-trace stock remove <code>`

移除股票。

```bash
$ fund-trace stock remove 600519
$ fund-trace stock remove sh 600519
```

### `fund-trace history <code> [--days N]`

历史净值 + 技术分析（仅限基金）。

```bash
$ fund-trace history 011513 --days 60
```

> 股票历史暂未实现，调用会返回明确错误提示。

### `fund-trace alert set <code> --drop 3`

设置告警阈值。也支持 `--rise`。

```bash
$ fund-trace alert set 011513 --drop 3
```

### `fund-trace alert list`

列出所有告警。

### `fund-trace alert remove <id>`

按 ID 移除告警。

### `fund-trace export [--format csv|html]`

导出基金和股票实时数据。

```bash
$ fund-trace export -f csv
$ fund-trace export -f html
```

CSV 列：Type, Market, Code, Name, Price/NAV, Previous, Change%, UpdateTime

## 配置文件

支持两种格式，推荐使用新的 `assets:` 格式。旧 `funds:` 格式自动兼容。

### 推荐格式（新）

```yaml
assets:
  - kind: fund
    code: "011513"
  - kind: stock
    market: sh
    code: "600519"
  - kind: stock
    market: sz
    code: "000001"

settings:
  refresh_interval_sec: 60
  cache_ttl_min: 6
  alert_cooldown_min: 30
  max_concurrent_requests: 5
```

### 兼容格式（旧，自动迁移）

```yaml
funds:
  - code: "011513"
  - code: "011925"
```

TUI 中增删操作自动写回 `assets:` 格式，已保存的旧配置在下次保存时自动升级。

## 股票市场推断规则

| 代码前缀 | 市场 | 说明 |
|---------|------|------|
| 6 | sh（上海） | 主板 |
| 0 | sz（深圳） | 主板 |
| 3 | sz（深圳） | 创业板 |
| 4 | — | 北交所（暂未支持） |
| 8 | — | 北交所（暂未支持） |

## 数据源

| 数据源 | 用途 |
|---|---|
| 天天基金 `fundgz.1234567.com.cn` | 基金实时估值 |
| 东方财富 `api.fund.eastmoney.com` | 基金历史净值 |
| 东方财富 `fund.eastmoney.com/js/fundcode_search.js` | 基金名称发现 |
| 腾讯财经 `qt.gtimg.cn` | A 股实时行情 |

> ⚠️ 腾讯财经接口为公开非正式接口，仅用于个人行情查看，不保证长期稳定，不构成投资建议。
> 天天基金自 2022 年起仅对指数型基金提供实时估值，QDII、混合型等基金实时估值不可用，系统会正确显示基金名称但净值列标记为 `—`。

## 项目结构

```
fund-trace/
├── main.go
├── go.mod / go.sum
├── config.yaml
└── internal/
    ├── model/       # Asset, Quote, Fund, RealTimeFund, NavSnapshot, Alert
    ├── config/      # YAML 解析、校验、Save（支持新旧格式）
    ├── store/       # SQLite WAL，assets/funds 表 CRUD
    ├── fetcher/     # 并发 API 客户端（天天基金/东方财富/腾讯财经）
    ├── analysis/    # SMA, EMA, RSI, TrendSummary
    ├── notifier/    # 桌面通知，冷却去重
    ├── tui/         # Bubble Tea 仪表盘，Sparkline，模态弹窗
    └── cmd/         # Cobra 命令行定义
```

## 构建

```bash
go build -o fund-trace .
```

要求 Go 1.22+。纯 Go 实现，无需 CGO。

## 已知限制

- 股票详情页历史分析暂未实现，详情页仅显示当前行情
- 股票告警暂未支持，选中股票时设置告警给出明确提示
- 北交所股票（代码 4/8 开头）暂未支持
- 港股、美股暂未支持
- 腾讯财经接口不保证长期稳定
