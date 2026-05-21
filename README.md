# fund-trace

中国公募基金实时跟踪终端工具。纯 Go 单二进制，SQLite 持久化，Bubble Tea TUI 仪表盘。下载即用，零外部依赖。

## 特性

**数据获取**
- 并发拉取天天基金实时估值，13 只基金同时请求，1-2 秒内返回
- 自动发现基金名称——只需提供 6 位代码，系统从东方财富基金数据库匹配全称
- 支持历史净值查询，默认 30 天，可自定义天数

**界面**
- TUI 交互仪表盘，启动即进入全屏终端界面，自动刷新
- 涨跌红绿着色——正值绿色、负值红色、零值灰色
- 迷你趋势线，基于最近 30 天历史净值渲染 Unicode 柱状图
- CLI 模式同样支持彩色输出，适配脚本和管道场景

**持久化**
- SQLite WAL 模式存储，崩溃安全，读写性能优于默认回滚日志
- 四张表：`funds`（基金元数据）、`nav_snapshots`（每日净值快照）、`alerts`（告警规则）、`daily_summary`（每日汇总）
- 首次运行自动建库建表，种子数据来自 `config.yaml`

**分析**
- SMA(5/10/20) 简单移动平均——短期/中期趋势判断
- EMA 指数移动平均——近期价格权重更高
- RSI(14) 相对强弱指数——超买 (>70) / 超卖 (<30) 信号
- 趋势方向判定：SMA5 vs SMA20 交叉

**告警**
- 支持跌幅告警（`--drop 3`）和涨幅告警（`--rise 5`）
- 桌面通知（macOS 通知中心，Linux/Windows 对应系统通知）
- 冷却机制——同一告警在冷却期内不重复推送，避免骚扰

**导出**
- CSV 格式——Excel/Numbers 可直接打开
- HTML 格式——带内联样式的自包含页面，涨跌着色

## 快速开始

```bash
# 下载对应平台的二进制文件
# macOS (Apple Silicon): fund-trace-darwin-arm64
# macOS (Intel):        fund-trace-darwin-amd64
# Linux:                fund-trace-linux-amd64

# 赋予执行权限后运行
chmod +x fund-trace
./fund-trace
```

首次运行自动创建 `fund-trace.db` 数据库文件，加载 `config.yaml` 中的基金列表。按 `q` 退出 TUI 界面。

如需修改跟踪的基金，编辑 `config.yaml` 中的 `funds` 列表，或在运行时使用 `add`/`remove` 命令动态管理。

## 命令参考

### `fund-trace`（默认）

启动全屏 TUI 仪表盘。自动刷新（默认 60 秒间隔），显示所有跟踪基金的实时估值、涨跌幅、趋势线。

```
基金 Trace                           2026-05-21 14:35:22

 Code      Name                      NAV         Change %      Trend      
────────────────────────────────────────────────────────────────────────
001595    天弘中证银行ETF联接C      1.6445      +0.22%        ▁▃▅▆█    
011513    天弘中证新能源车C         1.3317      -1.40%        ▅▄▃▂▁    

Last update: 14:35:22 | Next refresh: 54s
[q]uit  [r]efresh
```

### `fund-trace list`

以表格形式列出所有跟踪基金的当前实时数据，适合脚本和快速查看。

```bash
$ fund-trace list
```

### `fund-trace add <code>`

添加一只新基金。系统自动从东方财富基金数据库查找对应的基金全称。

```bash
$ fund-trace add 000001
Added fund 000001: 华夏成长混合
```

### `fund-trace remove <code>`

从跟踪列表中移除一只基金。别名：`rm`。

```bash
$ fund-trace remove 000001
Removed fund 000001
```

### `fund-trace history <code> [--days N]`

查看基金历史净值，附带技术分析指标。

```bash
$ fund-trace history 011513 --days 60

=== History: 011513 (60 days) ===

Date         NAV    Change%
─────────────────────────────────────
2026-05-20   1.3400 +1.52%
2026-05-19   1.3200 -0.75%
...

=== Trend Analysis ===
  Direction:   down
  5-day change: -2.13%
  SMA(5):      1.3250
  SMA(20):     1.3800
  RSI(14):     42.35 (neutral)
```

数据优先从本地 SQLite 缓存读取；缓存不足时自动从东方财富 API 拉取并存入本地。

### `fund-trace alert set <code> --drop 3`

设置跌幅告警。当日跌幅达到或超过 3% 时推送桌面通知。也支持 `--rise` 设置涨幅告警。

```bash
$ fund-trace alert set 011513 --drop 3
Alert #1 set: 011513 will notify on 3.0% drop

$ fund-trace alert set 007531 --rise 5
Alert #2 set: 007531 will notify on 5.0% rise
```

### `fund-trace alert list`

列出所有已配置的告警规则。

```bash
$ fund-trace alert list

=== Configured Alerts ===

ID  Code     Type   Threshold  Status
──────────────────────────────────────────
1   011513   drop    -3.0%     active
2   007531   rise    +5.0%     active
```

### `fund-trace alert remove <id>`

按 ID 移除一条告警规则。别名：`rm`。

```bash
$ fund-trace alert remove 1
Removed alert #1
```

### `fund-trace export [--format csv|html]`

导出当前实时数据为 CSV 或 HTML 文件。文件名自动带日期戳（如 `fund-data-2026-05-21.csv`）。

```bash
$ fund-trace export -f csv
Exported to fund-data-2026-05-21.csv

$ fund-trace export -f html
Exported to fund-data-2026-05-21.html
```

### `fund-trace monitor`

`fund-trace` 的别名，同样启动 TUI 仪表盘。支持缩写 `mon`。

```bash
$ fund-trace mon
```

## 配置文件

`config.yaml` 是全局配置入口，必须与二进制文件放在同一目录，或通过 `-c` 指定路径。

```yaml
funds:
  - code: "011513"   # 天弘中证新能源车C
  - code: "011925"   # 嘉实港股互联网产业核心资产C
  - code: "017435"   # 华宝中证沪港深新消费指数C
  - code: "012734"   # 易方达中证人工智能主题ETF联接C
  - code: "008087"   # 华夏中证5G通信主题ETF联接C
  - code: "011609"   # 易方达上证科创50联接C
  - code: "012349"   # 天弘恒生科技ETF联接C
  - code: "007531"   # 华宝券商ETF联接C
  - code: "001595"   # 天弘中证银行ETF联接C
  - code: "016068"   # 鹏华新能源汽车混合C
  - code: "021492"   # 中航远见领航混合发起C
  - code: "562500"   # 机器人ETF华夏
  - code: "024913"   # 华夏国证通用航空产业ETF发起式联接C

settings:
  refresh_interval_sec: 60   # TUI 自动刷新间隔（秒）
  cache_ttl_min: 6            # API 缓存有效期（分钟），暂未启用
  alert_cooldown_min: 30      # 同一告警规则的最小触发间隔（分钟）
  max_concurrent_requests: 5  # 并发 API 请求数上限
```

**配置项说明：**

| 字段 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `funds[].code` | string | — | 6 位基金代码，必填 |
| `refresh_interval_sec` | int | 60 | TUI 面板刷新间隔。值过小会增加 API 请求频率 |
| `cache_ttl_min` | int | 6 | API 数据内存缓存有效期（预留字段） |
| `alert_cooldown_min` | int | 30 | 两分钟内同一只基金的同一类告警只触发一次 |
| `max_concurrent_requests` | int | 5 | 并发请求数上限。受信号量控制，防止瞬间大量连接 |

启动时 `config.yaml` 中的基金列表会被写入 SQLite 数据库。后续通过 `fund-trace add` 或 `fund-trace remove` 更新的基金信息会同步写入数据库，但不会回写到 YAML 文件——`config.yaml` 始终保持为手动编辑的源文件。

## 构建

```bash
git clone <repo-url>
cd fund-trace
go build -o fund-trace .
```

要求 Go 1.22 或更高版本。构建产物为静态链接的单一二进制文件（macOS 约 15MB）。

交叉编译其他平台：

```bash
GOOS=linux  GOARCH=amd64 go build -o fund-trace-linux-amd64  .
GOOS=linux  GOARCH=arm64 go build -o fund-trace-linux-arm64  .
GOOS=windows GOARCH=amd64 go build -o fund-trace-windows.exe .
```

所有依赖使用纯 Go 实现（`modernc.org/sqlite`），无需 CGO、无需安装任何系统库。

## 技术栈

| 组件 | 库 |
|---|---|
| CLI 框架 | `github.com/spf13/cobra` |
| TUI 仪表盘 | `github.com/charmbracelet/bubbletea` |
| 终端样式 | `github.com/charmbracelet/lipgloss` |
| SQLite 驱动 | `modernc.org/sqlite`（纯 Go，零 CGO） |
| 桌面通知 | `github.com/gen2brain/beeep` |
| YAML 配置 | `gopkg.in/yaml.v3` |
| 测试 | Go 标准库 `testing` + `httptest`（模拟 API） |

## 数据源

本工具使用两个公开的中国金融数据 API，无需任何 API Key：

| 数据源 | 用途 | 接口地址 |
|---|---|---|
| 天天基金 | 实时估值（盘中净值估算） | `fundgz.1234567.com.cn/js/{code}.js` |
| 东方财富 | 历史净值（每日单位净值） | `api.fund.eastmoney.com/f10/lsjz` |
| 东方财富 | 基金名称发现（全量基金列表） | `fund.eastmoney.com/js/fundcode_search.js` |

**注意事项：**
- 天天基金自 2022 年起仅对指数型基金提供实时估值数据，非指数型基金（混合型、债券型等）的实时估值字段可能为空，系统会显示 `—`
- 历史净值数据来自东方财富，通常包含过去 30 天至数年的每日净值记录
- 两个接口均无官方速率限制，但本工具内置信号量控制并发数，避免对服务器造成压力
- 所有数据仅供个人参考，不构成投资建议

## 项目结构

```
fund-trace/
├── main.go                   # 入口（3行）
├── go.mod / go.sum           # 依赖管理
├── config.yaml               # 基金列表与全局配置
├── README.md
└── internal/
    ├── model/                # 数据模型（Fund, RealTimeFund, NavSnapshot, Alert）
    ├── config/               # YAML 配置解析与校验
    ├── store/                # SQLite 持久化（WAL 模式，4 张表，事务批量写入）
    ├── fetcher/              # 并发 API 客户端（信号量限流、指数退避重试、JSONP 解析）
    ├── analysis/             # 技术指标计算（SMA, EMA, RSI, 趋势判定）
    ├── notifier/             # 桌面通知（beeep，带冷却去重）
    ├── tui/                  # Bubble Tea 终端仪表盘（自动刷新、CJK 对齐、迷你趋势线）
    └── cmd/                  # Cobra CLI 命令定义（7 个子命令）
```

## 许可

MIT
