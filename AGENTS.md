# AGENTS.md

本仓库面向 AI 代理（以及人类贡献者）的开发指南。

## 项目简介

Wails v3 桌面应用，用于把安卓版《我在地府打麻将》的存档迁移到
Windows 上的 Steam 版本。**本项目仅支持 Windows**，CI、打包、构建
脚本均只针对 Windows。不要重新添加其他平台的构建产物（Android /
iOS / macOS / Linux / Docker 跨编译），除非明确要支持该平台。

- **后端**: Go 1.26（`go.mod` 要求 `go 1.26.0`），Wails v3 (`github.com/wailsapp/wails/v3 v3.1.0`)
- **前端**: Vue 3 + TypeScript + Vite，包管理器用 `pnpm`（不要用 `npm` / `yarn`）
- **存档库**: SQLite（`modernc.org/sqlite`，纯 Go 驱动，无 CGO）

## 目录结构

```
.
├── main.go                      # 入口：创建 app、注册服务、打开窗口
├── internal/
│   ├── android/
│   │   ├── android.go           # Service：AutoFetch / ReadSlots / Wi-Fi / PickDB
│   │   ├── adb.go               # 内嵌 adb.exe、设备列表、拉取 game.db
│   │   ├── db.go                # 读取 SQLite 的 `slot` 表
│   │   ├── wifi.go              # 手机 Wi-Fi 上传用的 HTTP 服务
│   │   └── embedded/            # 内置 adb.exe + DLL（//go:embed）
│   ├── migrate/migrate.go       # Service：备份并写入 Slot{0,1,2}.json
│   └── steam/steam.go           # Service：注册表定位 Steam + 扫描 userdata
├── frontend/
│   ├── src/
│   │   ├── App.vue             # 顶层状态 + 流程编排
│   │   ├── components/         # AndroidPanel、SteamPanel、SlotTable、MigratePanel
│   │   └── main.ts
│   ├── bindings/               # `wails3 generate bindings -ts` 自动生成，已提交进仓库
│   └── package.json
├── build/
│   ├── config.yml              # Wails 项目配置（dev 模式、产品元数据）
│   ├── Taskfile.yml            # 前端构建 / bindings / icons 等通用任务
│   ├── appicon.png             # Windows 图标源图（.ico 由 generate:icons 生成）
│   └── windows/                # Windows Taskfile、NSIS 安装包、MSIX 模板、info.json、图标
├── Taskfile.yml                # 根任务入口（build / package / run / dev）
├── .github/workflows/          # ci.yml、lint.yml、release.yml（均在 windows-latest 运行）
└── .golangci.yaml
```

## 开发环境

- Go 1.26+（CI 固定 `1.26`；本机 `go` 可能是更高版本，见「常见坑」）
- Node.js 22+
- pnpm 11+
- golangci-lint v2.15+（需能解析本机 Go 标准库；CI 同版本）
- Wails v3 CLI（与 go.mod 中的库版本保持一致）：

  ```powershell
  go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
  ```

- `wails3` 内置 task runner，直接 `wails3 task <name>` 即可；本机另外安装 go-task 则可用 `task <name>`，效果相同。

## 常用命令

```powershell
# 开发模式（vite dev server + 热重载）
wails3 dev -config ./build/config.yml -port 9245
wails3 task dev          # 等效，经根 Taskfile

# 构建（生产构建；经 build/windows/Taskfile.yml 完成前端构建 + bindings + syso + 嵌入）
wails3 build
wails3 task build

# 打包 NSIS 安装包（默认）；MSIX：wails3 task package FORMAT=msix
wails3 task package
```

只构建前端：

```powershell
cd frontend
pnpm install
pnpm build      # vue-tsc --noEmit && vite build
pnpm lint
pnpm format
cd ..
```

提交前完整验证：

```powershell
wails3 generate bindings -ts    # 1. 改过 Go service 签名后必须重新生成并提交
cd frontend; pnpm build; cd ..   # 2. 先产出 frontend/dist（必须先于 go build）
go build .                       # 3. 嵌入最新前端产物
go test ./...
golangci-lint run
```

直接用 `wails3 build`（或 `wails3 task build`）可以保证上述顺序自动正确。

## Wails v3 关键约定

### 服务注册（关键模式）

Service 方法需要访问 `*application.App`（Dialog / Logger / Events 等）。
**先创建 app，再通过构造函数注入，然后注册**：

```go
app := application.New(application.Options{...})
app.RegisterService(application.NewService(android.NewService(app)))
```

不要注册零值结构体（如 `&android.Service{}`），否则 `app` 字段为 `nil`，
所有调用都会报 "app is not initialized"。

### Service 方法签名（强制）

所有暴露给前端的 service 方法必须使用 request/response 信封，即使载荷为空：

```go
func (s *Service) MethodName(req *MethodNameRequest) (*MethodNameResponse, error)
```

空的 request 结构体没关系：`type AutoFetchAndroidDBRequest struct{}`。

### 重新生成 bindings

修改 Go service 签名或 model 字段后：

```powershell
wails3 generate bindings -ts
```

- `-ts` 生成 TypeScript（`.ts`），前端 import 的就是它；不加 `-ts` 会生成 `.js`，破坏 `vue-tsc --noEmit`。
- v3.1.0 的生成器会输出 `frontend/bindings/events/index.ts` 等新文件，`-clean` 默认开启，会先清空输出目录再生成。
- 生成结果使用 `@wailsio/runtime` 的 `Call` / `Create.Nullable` 等 API；生成文件已提交进仓库，CI 不运行生成器。
- `frontend/bindings/` 有改动就一起提交，否则前端调用与 Go 签名不一致，编译或运行时会出错。

### 前端运行时依赖

`frontend/package.json` 中的 `@wailsio/runtime` 必须与 go.mod 中的 wails
主版本一致（当前均为 3.1.0，写法为 `^3.1.0`）。**不要用 `latest`**——
历史上 `latest` 导致运行时停留在 `3.0.0-beta.10` 而 Go 侧是 v3.1.0，两者 API 不匹配。

## 构建与嵌入

`main.go` 用 `//go:embed all:frontend/dist` 嵌入前端产物。因此
**必须先 `pnpm build`，再 `go build`**，否则二进制里嵌入的是旧前端，UI
会静默异常（比如状态消息不更新）。`wails3 build` 的任务链（windows:build
→ common:build:frontend → generate:bindings → pnpm run build → go build）
已经保证了顺序。

## Windows 平台特有事项

- 源码模块名是 `demonicmahjong-slot-migrate`，但发布产物名刻意保留
  `WodimaSlotMigrate.exe`（根 `Taskfile.yml` 的 `APP_NAME`），GitHub
  Release 上传的 `bin/*.exe` 因此不受改名影响。
- 运行时解压的 adb.exe 位于 `%LOCALAPPDATA%\demonicmahjong-migrate\`；
  旧的 `wodima-migrate` 目录可手动删除。
- Steam 路径通过注册表定位（`golang.org/x/sys/windows/registry`）。
- 打包走 `build/windows/`：`wails3 task package` 默认生成 NSIS 安装包
  （`FORMAT=msix` 则走 MSIX）。NSIS 需要 `makensis`；MSIX 需要
  `wails3 tool msix-install-tools`。
- `build/config.yml` 的 `info` 与 `build/windows/` 下的
  `info.json`、`wails.exe.manifest`、`nsis/wails_tools.nsh`、`msix/` 元数据
  描述的是同一个产品（名称 / 版本 / 公司名 / productIdentifier）。修改
  `info` 时这些文件要一起同步（`wails3 build` 会按 `info` 重新生成
  `wails_tools.nsh`，其余手动保持一致即可）。
- `wails_windows_*.syso` 是构建时生成的 Windows 资源文件，已 gitignore，正常。

## Wi-Fi 上传功能的防火墙说明

Wi-Fi 上传服务监听 `0.0.0.0:<随机端口>`。Windows 防火墙在「公用网络」
下默认拦截入站连接，手机端会报 `net::ERR_CONNECTION_ABORTED`。UI 里给
了两个修复方式：

1. `Set-NetConnectionProfile -NetworkCategory Private`（推荐）
2. `netsh advfirewall firewall add rule name="..." dir=in action=allow protocol=TCP localport=<port> profile=any`

遇到这类报错时，几乎都是网络配置文件的问题。

## Git 约定

- 使用 [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)：`feat:` `fix:` `refactor:` `docs:` `chore:` `build:` `ci:`。
- 提交信息用英文。
- 只有在用户明确要求时才提交；在默认分支上先建分支。

## 代码风格

- **Go**: `gofmt` + `gci`（import 分组），配置见 `.golangci.yaml`（强制 `interface{}` → `any`、行宽 ≤ 380 等）。
- **换行符**: `.gitattributes` 强制 LF（Go/TS/Vue/YAML）。`gci` 对 CRLF 敏感，而 Windows runner 上 `core.autocrlf=true` 会污染文件。
- **前端**: Prettier + ESLint + vue-tsc；Husky `pre-commit` 对暂存的前端文件跑 `lint-staged`。
- 注释使用英文，且只在逻辑不明显时写。

## PowerShell

用户的 shell 是 Windows 上的 PowerShell。生成命令必须是 PowerShell 兼容的（不用 `&&`，改用 `;`；换行用反引号；路径含空格要加引号）。

## CI

均在 `windows-latest` 上运行；Wails CLI 固定为 `v3.1.0`，Go 固定为 `1.26`。

- `.github/workflows/ci.yml` — 安装依赖、构建前端、`wails3 build`。
- `.github/workflows/lint.yml` — `golangci-lint`（先 `pnpm build` 保证 `//go:embed` 可解析；`skip-cache: true` 避免 bindings 重构后命中过期缓存）+ `pnpm lint`。
- `.github/workflows/release.yml` — `v*` 标签触发，`wails3 build` 后把 `bin/*.exe` 上传到 GitHub Release。

## 常见坑

1. **"app is not initialized"** — service 注册时用了零值结构体而不是 `NewService(app)`。见「服务注册」。
2. **`pattern all:frontend/dist: no matching files found`** — `frontend/dist` 被 gitignore。CI/lint 必须先 `pnpm build` 再 `go build` / `golangci-lint`。
3. **`gci: File is not properly formatted` 只在 CI 出现** — 换行符问题；确认 `.gitattributes` 已提交且覆盖该文件类型。
4. **bindings 不同步** — 改了 Go 签名后运行 `wails3 generate bindings -ts` 并一起提交，否则 `vue-tsc` 失败或调用静默错位。
5. **v3.1.0 没有 `-webview2` 构建参数** — `wails3 build` 直接可用；NSIS 安装包走 `wails3 task package`（会自动拉取 WebView2 bootstrapper）。
6. **前端嵌入了旧版本** — `go build .` 嵌入的是 `frontend/dist` 当时的内容；先 `pnpm build` 再 `go build`，或直接用 `wails3 build`。
7. **golangci-lint 与 Go 工具链不匹配** — golangci-lint 必须能解析本机 Go 的标准库：老版本（v2.13，go1.26 构建）在 go1.27 下直接 panic。本机与 CI 均使用 v2.15。如升级本机 Go 大版本后 lint 异常，先升级 golangci-lint（`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`）。
8. **`@wailsio/runtime` 版本漂移** — 前端依赖不要用 `latest` 标签，见「前端运行时依赖」。
9. **productIdentifier / 名称不一致** — 改 `build/config.yml` 的产品元数据时，记得同步 `build/windows/` 下的图标、清单、NSIS / MSIX 元数据和版本号。
10. **CI runner 里 `sh: pnpm --version` 报 "pnpm not found"** — `pnpm/setup` 只提供 `.cmd` / `.ps1` shim，go-task 内置 shell（mvdan/sh）不按 PATHEXT 解析，只能精确匹配文件名。Taskfile 里调用 pnpm（及其他 .cmd/.ps1 形式的 shim）一律用 `cmd:`（走系统 shell）。
11. **`wails3 generate icons` 默认还会生成 mac 图标**（`-macfilename` 默认 `build/darwin/icon.icns`）— 仅 Windows 项目需显式 `-macfilename ""` 跳过，否则报 `open build/darwin/icon.icns: ...`。
