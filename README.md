# fund-trace

中国公募基金实时跟踪终端工具。纯 Go 单二进制，SQLite 持久化，Bubble Tea TUI 交互仪表盘。下载即用，零外部依赖。

## 特性

**TUI 交互操作**
- 光标导航（j/k / ↑↓），选中行反显高亮
- `a` 添加基金：输入 6 位代码，自动发现基金名称，立即生效并写入配置
- `d` 删除基金：二次确认后移除，同步更新配置文件
- `A` 设置涨跌告警：切换跌/涨类型，输入阈值百分比
- `s` 打开设置面板：实时编辑刷新间隔、并发数等参数，Esc 保存到 config.yaml
- `Enter` 查看基金详情：全屏展示 SMA/RSI 趋势分析 + 最近 10 天历史净值
- `h` 快捷键帮助浮层，随时查看所有可用操作

**数据获取**
- 并发拉取天天基金实时估值，所有基金同时请求，1-2 秒内返回
- 自动发现基金名称——只需 6 位代码，系统从东方财富 26400+ 基金数据库中匹配全称
- 启动时自动补齐配置文件中缺失的基金名称
- 支持历史净值查询，默认 30 天，可自定义天数

**Sparkline 趋势线**
- 0% 锚定绝对坐标系：▁▂▃▄▅▆▇█，0% 固定在 ▄，正数往上、负数往下
- 数据源为日涨跌幅，每个块直接反映当天涨跌幅度
- 逐块独立染色：正值绿色、负值红色、▄（0% 附近）灰色
- 最右块自动追加当日实时估值，始终与 Change% 列同步

**持久化**
- SQLite WAL 模式存储，崩溃安全
- 四张表：`funds`、`nav_snapshots`、`alerts`、`daily_summary`
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
- CSV / HTML 双格式，文件名自动带日期戳

## 快速开始

从 [Releases](https://github.com/xijuangu/fund-trace/releases) 下载对应平台的二进制文件。

```bash
# macOS / Linux
chmod +x fund-trace-darwin-arm64
./fund-trace-darwin-arm64

# Windows（在 Windows Terminal 或 PowerShell 中运行）
.\fund-trace-windows-amd64.exe
```

首次运行时若没有 `config.yaml`，会自动生成一份包含 13 只默认基金的配置文件。`fund-trace.db` 也会同时创建在当前目录。

## TUI 快捷键

| 键 | 功能 |
|---|---|
| `j` `k` `↑` `↓` | 移动光标，选中行反显高亮 |
| `a` | 添加基金（输代码 → 回车确认，自动发现名称） |
| `d` | 删除选中基金（`y` 确认 / `n` 取消） |
| `A` | 为选中基金设置告警（`t` 切换跌/涨，输阈值 → 回车） |
| `s` | 设置面板（`j`/`k` 选字段，回车编辑，Esc 保存并退出） |
| `Enter` | 查看选中基金详情：趋势分析 + 历史净值表 |
| `h` | 帮助浮层（任意键关闭） |
| `r` | 手动刷新数据 |
| `q` `Esc` | 退出（弹窗模式下 Esc = 关闭弹窗） |

## 命令参考

### `fund-trace`（默认）

启动全屏 TUI 交互仪表盘。

```
 Fund Trace                           2026-05-22 10:30:00

 Code      Name                      NAV         Change %      Trend       
 ────────────────────────────────────────────────────────────────────────
 001595    天弘中证银行ETF联接C      1.6450      +0.04%        ▄▄▄▂▂▄▄▄▃▃ 
 008087    华夏中证5G通信主题ETF联…  3.1259      +2.77%        ▆▂▄▄▆▅▆▂▄▂ 

 Last update: 10:30:00 | Next refresh: 52s
 [j/k]nav  [Enter]detail  [a]dd  [d]el  [A]lert  [s]ettings  [h]elp  [r]efresh  [q]uit
```

### `fund-trace list`

表格形式列出所有基金的当前实时数据。

```bash
$ fund-trace list
```

### `fund-trace add <code>`

添加基金，自动发现名称。

```bash
$ fund-trace add 000001
Added fund 000001: 华夏成长混合
```

### `fund-trace remove <code>`

移除基金。别名：`rm`。

```bash
$ fund-trace remove 000001
Removed fund 000001
```

### `fund-trace history <code> [--days N]`

历史净值 + 技术分析。

```bash
$ fund-trace history 011513 --days 60

=== History: 011513 (60 days) ===

Date         NAV      Change%
────────────────────────────────
2026-05-20   1.3400   +1.52%
2026-05-19   1.3200   -0.75%
...

=== Trend Analysis ===
  Direction:   down
  5-day change: -2.13%
  SMA(5):      1.3250
  SMA(20):     1.3800
  RSI(14):     42.35 (neutral)
```

### `fund-trace alert set <code> --drop 3`

设置告警阈值。也支持 `--rise`。

```bash
$ fund-trace alert set 011513 --drop 3
Alert #1 set: 011513 will notify on 3.0% drop
```

### `fund-trace alert list`

列出所有告警。

```bash
$ fund-trace alert list

=== Configured Alerts ===

ID  Code     Type   Threshold  Status
──────────────────────────────────────────
1   011513   drop    -3.0%     active
```

### `fund-trace alert remove <id>`

按 ID 移除告警。别名：`rm`。

```bash
$ fund-trace alert remove 1
```

### `fund-trace export [--format csv|html]`

导出实时数据。

```bash
$ fund-trace export -f csv   # → fund-data-2026-05-22.csv
$ fund-trace export -f html  # → fund-data-2026-05-22.html
```

### `fund-trace monitor`

TUI 仪表盘的别名，支持缩写 `mon`。

## 配置文件

`config.yaml` 与二进制放在同一目录，或用 `-c` 指定路径。首次运行不存在时自动生成。

```yaml
funds:
  - code: "011513"
  - code: "011925"

settings:
  refresh_interval_sec: 60   # TUI 刷新间隔（秒）
  cache_ttl_min: 6            # API 缓存有效期（分钟）
  alert_cooldown_min: 30      # 同一告警最小触发间隔（分钟）
  max_concurrent_requests: 5  # 并发请求数上限
```

TUI 中的增删操作会自动写回 `config.yaml`，无需手动编辑。

## 文件位置

| 文件 | 默认路径 | 用途 |
|---|---|---|
| `config.yaml` | `./config.yaml` | 基金列表与全局配置 |
| `fund-trace.db` | `./fund-trace.db` | SQLite 数据库 |

均在执行目录下生成。可通过 `-c` 指定自定义配置文件路径。

## 构建

```bash
git clone https://github.com/xijuangu/fund-trace.git
cd fund-trace
go build -o fund-trace .
```

要求 Go 1.22+。纯 Go 实现，无需 CGO。交叉编译：

```bash
GOOS=linux  GOARCH=amd64 go build -o fund-trace-linux-amd64  .
GOOS=windows GOARCH=amd64 go build -o fund-trace-windows.exe .
```

## 技术栈

| 组件 | 库 |
|---|---|
| CLI | `spf13/cobra` |
| TUI | `charmbracelet/bubbletea` + `bubbles/textinput` |
| 样式 | `charmbracelet/lipgloss` |
| SQLite | `modernc.org/sqlite`（纯 Go） |
| 通知 | `gen2brain/beeep` |
| 配置 | `gopkg.in/yaml.v3` |

## 数据源

无需 API Key。

| 数据源 | 用途 |
|---|---|
| 天天基金 `fundgz.1234567.com.cn` | 实时估值 |
| 东方财富 `api.fund.eastmoney.com` | 历史净值 |
| 东方财富 `fund.eastmoney.com/js/fundcode_search.js` | 基金名称发现（26400+ 条） |

> 天天基金自 2022 年起仅对指数型基金提供实时估值，QDII、混合型等基金实时估值不可用，系统会正确显示基金名称但净值列标记为 `—`。所有数据仅供个人参考，不构成投资建议。

## 项目结构

```
fund-trace/
├── main.go
├── go.mod / go.sum
├── config.yaml
└── internal/
    ├── model/       # Fund, RealTimeFund, NavSnapshot, Alert
    ├── config/      # YAML 解析、校验、Save
    ├── store/       # SQLite WAL, 4 表 CRUD
    ├── fetcher/     # 并发 API 客户端, 信号量, 重试
    ├── analysis/    # SMA, EMA, RSI, TrendSummary
    ├── notifier/    # 桌面通知, 冷却去重
    ├── tui/         # Bubble Tea 仪表盘, Sparkline, 模态弹窗
    └── cmd/         # Cobra 命令行定义
```

## 许可

MIT
