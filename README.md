# 我在地府打麻将 · 存档迁移

一个 Windows 桌面工具（[Wails v3](https://v3.wails.io/) 构建），用于把安卓版《我在地府打麻将》的存档迁移到 Windows 上的 Steam 版本。

![应用截图](images/app.png)

> 本项目**仅支持 Windows**（存档迁移依赖 Windows 注册表定位 Steam、依赖内置的 Windows 版 `adb.exe`）。CI、打包流程（NSIS / MSIX）也均只针对 Windows。

## 功能

- **自动定位 Steam 存档目录** — 通过注册表找到 Steam 安装路径，遍历 `userdata`，定位到 appid `3444020` 对应的存档目录
- **三种安卓存档获取方式**
  - ADB 自动拉取（需 USB 调试，内置 adb.exe，无需单独安装）
  - 手动选择 `game.db` 文件
  - 同一局域网内 Wi-Fi 上传（手机扫码即可）
- **存档预览与选择性迁移** — 读取 SQLite 的 `slot` 表，列出全部存档，勾选要迁移的项
- **安全迁移** — 迁移前自动把原存档备份为 `.bak` 文件

## 工作流程

1. 选择 Steam 存档目录（自动检测 / 手动指定）
2. 获取安卓存档（ADB / 手动选择 / Wi-Fi 上传）
3. 预览存档列表，勾选要迁移的存档
4. 点击「开始迁移」，完成

## 环境要求

- [Go](https://go.dev/) 1.26+
- [Node.js](https://nodejs.org/) 22+
- [pnpm](https://pnpm.io/) 11+
- Wails v3 CLI（版本需与 `go.mod` 中的 wails 库一致）：

  ```powershell
  go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
  ```

## 开发

```powershell
# 安装前端依赖并启动开发模式（vite dev server + 热重载）
cd frontend; pnpm install; cd ..
wails3 dev -config ./build/config.yml -port 9245
# 或者：wails3 task dev
```

修改 Go service 签名或模型后，重新生成前端绑定并提交：

```powershell
wails3 generate bindings -ts
```

构建与测试：

```powershell
wails3 build          # 生产构建（自动先构建前端、生成 bindings、再编译嵌入）
wails3 task build     # 等效写法
go test ./...
```

打包（需要 `makensis` 或 MSIX 工具）：

```powershell
wails3 task package              # NSIS 安装包（默认）
wails3 task package FORMAT=msix  # MSIX 包
```

发布版本：推送 `v*` 标签即可触发 GitHub Actions，自动构建并把 `bin/*.exe` 上传到 Release。

## 项目结构

```
.
├── main.go                # 入口：创建 Wails 应用、注册 service、打开窗口
├── internal/
│   ├── android/           # 安卓存档获取（ADB / Wi-Fi / SQLite 解析 / 内嵌 adb.exe）
│   ├── migrate/           # 迁移核心：备份原存档 + 写入 Slot{0,1,2}.json
│   └── steam/             # Steam 安装路径（注册表）与 userdata 扫描
├── frontend/              # Vue 3 + TypeScript 前端
│   └── bindings/          # wails3 自动生成的绑定（已提交）
├── build/
│   ├── config.yml         # Wails 项目配置与产品元数据
│   ├── Taskfile.yml       # 通用任务（前端构建、bindings、图标）
│   └── windows/           # Windows Taskfile、NSIS / MSIX 打包、图标、info.json
├── Taskfile.yml           # 根任务入口（dev / build / package / run）
└── .github/workflows/     # ci / lint / release（windows-latest）
```

## 常见问题

**手机扫码上传失败（`net::ERR_CONNECTION_ABORTED`）**
通常是 Windows 防火墙在「公用网络」下拦截了入站连接。把当前网络设为「专用」：

```powershell
Set-NetConnectionProfile -NetworkCategory Private
```

## 许可证

[MIT](LICENSE)
