<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import QRCode from 'qrcode'

const props = defineProps<{
  dbPaths: string[]
  busy: boolean
  status: string
  error: string
  wifiUrl: string
  wifiLocalUrl: string
  wifiAllUrls: string[]
  wifiActive: boolean
  wifiWaiting: boolean
  wifiError: string
  wifiStatus: string
  wifiDebugInfo: string
  wifiFirewallCmd: string
}>()

const showDebugInfo = ref(false)

// Wi-Fi feature flag - disabled until fully implemented
const WIFI_ENABLED = true

// Filter out localhost from URL list for LAN-only display
const lanUrls = computed(() => props.wifiAllUrls.filter((url) => !url.includes('localhost')))

const emit = defineEmits<{
  (_e: 'auto'): void
  (_e: 'pick'): void
  (_e: 'wifi'): void
}>()

// Android device paths (constants, shown for reference).
// Standard install plus TapTap sandboxed install.
const ANDROID_DB_PATHS = [
  '/sdcard/Android/data/com.itaotuo.wodima/files/game.db',
  '/sdcard/Android/data/com.taptap/files/tap_sandbox_sd/0/Android/data/com.itaotuo.wodima/files/game.db',
]

// QR code SVG generated from wifiUrl
const qrSvg = ref('')

// Generate QR code when wifiUrl changes
watch(
  () => props.wifiUrl,
  async (url: string) => {
    if (url) {
      try {
        qrSvg.value = await QRCode.toString(url, { type: 'svg', width: 200 })
      } catch {
        qrSvg.value = ''
      }
    } else {
      qrSvg.value = ''
    }
  },
  { immediate: true },
)

async function copyCmd(cmd?: string) {
  const text = cmd || props.wifiFirewallCmd
  if (text) {
    try {
      await navigator.clipboard.writeText(text)
      alert('命令已复制到剪贴板！')
    } catch {
      alert('复制失败，请手动复制')
    }
  }
}
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>2. 安卓存档文件 (game.db)</h2>
    </header>
    <div class="panel-body">
      <!-- Android path reference -->
      <div class="row">
        <span class="label">手机端路径：</span>
        <div class="path-list">
          <div v-for="(p, i) in ANDROID_DB_PATHS" :key="i" class="path-item">
            <code class="path">{{ p }}</code>
            <span v-if="i === 0" class="badge-tag">独立安装包</span>
            <span v-else class="badge-tag alt">TapTap 启动</span>
          </div>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="row">
        <button class="btn" :disabled="props.busy" @click="emit('auto')">自动从手机获取</button>
        <button class="btn" :disabled="props.busy" @click="emit('pick')">手动选择 game.db</button>
        <button
          v-if="WIFI_ENABLED"
          class="btn"
          :disabled="props.busy || props.wifiActive"
          @click="emit('wifi')"
        >
          {{ props.wifiActive ? '停止 Wi-Fi 传输' : props.busy ? '启动中…' : '通过 Wi-Fi 获取' }}
        </button>
      </div>
      <p class="hint note">
        💡 「自动从手机获取」需要使用<strong>数据线</strong>连接手机，并在手机上开启
        <strong>USB 调试模式</strong>。
      </p>

      <!-- Status/Error messages -->
      <p v-if="props.status" class="hint">
        {{ props.status }}
      </p>
      <p v-if="props.error" class="error">
        {{ props.error }}
      </p>
      <p v-if="WIFI_ENABLED && props.wifiStatus" class="hint wifi-status">
        {{ props.wifiStatus }}
      </p>
      <p v-if="WIFI_ENABLED && props.wifiError" class="error">Wi-Fi 错误：{{ props.wifiError }}</p>

      <!-- Selected local file paths -->
      <div v-if="props.dbPaths.length" class="row">
        <span class="label">已加载文件：</span>
        <div class="path-list">
          <code v-for="(p, i) in props.dbPaths" :key="i" class="path">{{ p }}</code>
        </div>
      </div>

      <!-- Wi-Fi transfer panel -->
      <div v-if="WIFI_ENABLED && props.wifiActive" class="wifi-panel">
        <div v-if="props.wifiUrl" class="wifi-info">
          <p>📱 在手机上扫描二维码，或在浏览器打开：</p>
          <code class="url">{{ props.wifiUrl }}</code>
        </div>

        <div v-if="qrSvg" class="qr-container" v-html="qrSvg" />

        <p class="hint">
          请先将手机连入同一 Wi-Fi，在手机文件管理器中把 game.db
          复制到「Download」文件夹，然后在打开的网页上选择文件并上传。
        </p>

        <!-- All available URLs (excluding localhost) -->
        <div v-if="lanUrls.length > 1" class="all-urls">
          <p class="section-title">🔗 所有可用地址（逐个尝试）：</p>
          <ul class="url-list">
            <li
              v-for="(url, idx) in lanUrls"
              :key="idx"
              :class="{ primary: url === props.wifiUrl }"
            >
              <code class="url">{{ url }}</code>
              <span v-if="url === props.wifiUrl" class="badge">推荐</span>
            </li>
          </ul>
        </div>

        <div v-if="props.wifiFirewallCmd" class="firewall-section">
          <p class="section-title">🛡️ 防火墙配置（如手机无法访问）：</p>
          <p class="hint">以<strong>管理员身份</strong>打开 PowerShell，依次尝试以下方法：</p>

          <p class="sub-title">方法 1：将网络配置文件改为「专用」（推荐）</p>
          <p class="hint">
            如果你的 Wi-Fi 是「公用」网络，Windows 会阻止所有入站连接。将网络改为「专用」即可：
          </p>
          <div class="cmd-container">
            <code class="cmd">Set-NetConnectionProfile -NetworkCategory Private</code>
            <button
              class="btn-copy"
              @click="copyCmd('Set-NetConnectionProfile -NetworkCategory Private')"
            >
              复制
            </button>
          </div>

          <p class="sub-title">方法 2：添加端口规则</p>
          <div class="cmd-container">
            <code class="cmd">{{ props.wifiFirewallCmd }}</code>
            <button class="btn-copy" @click="copyCmd(props.wifiFirewallCmd)">复制</button>
          </div>
        </div>
        <div v-if="props.wifiDebugInfo" class="debug-section">
          <button class="btn-debug" @click="showDebugInfo = !showDebugInfo">
            {{ showDebugInfo ? '隐藏' : '显示' }}网络接口信息
          </button>
          <pre v-if="showDebugInfo" class="debug-info">{{ props.wifiDebugInfo }}</pre>
        </div>
        <button class="btn btn-stop" :disabled="props.busy" @click="emit('wifi')">
          停止 Wi-Fi 传输
        </button>
        <p v-if="props.wifiWaiting" class="hint">等待手机上传文件中…</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.wifi-status {
  color: #667eea;
  font-weight: 500;
}
.wifi-panel {
  margin-top: 16px;
  padding: 16px;
  background: var(--color-bg-elevated);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}
.wifi-info {
  text-align: center;
}
.wifi-info .url {
  display: block;
  margin: 8px 0;
  font-size: 13px;
  word-break: break-all;
}
.hint.note {
  margin-top: 8px;
  padding: 8px 10px;
  background: rgba(102, 126, 234, 0.1);
  border-radius: 6px;
  border-left: 3px solid var(--color-primary);
  color: var(--color-text);
  line-height: 1.5;
}
.qr-container {
  display: flex;
  justify-content: center;
  margin: 16px 0;
}
.qr-container svg {
  width: 200px;
  height: 200px;
}
.btn-stop {
  display: block;
  margin: 0 auto;
  background: var(--color-danger);
}
.btn-stop:hover {
  opacity: 0.9;
}
.debug-section {
  margin-top: 12px;
  text-align: center;
}
.btn-debug {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text-muted);
  padding: 4px 12px;
  font-size: 12px;
  border-radius: 4px;
  cursor: pointer;
}
.btn-debug:hover {
  color: var(--color-text);
  border-color: var(--color-text-muted);
}
.debug-info {
  text-align: left;
  margin: 8px auto 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  max-width: 500px;
  overflow-x: auto;
}
.all-urls {
  margin-top: 16px;
  padding: 12px;
  background: rgba(102, 126, 234, 0.05);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}
.all-urls .section-title {
  font-weight: 600;
  margin: 0 0 8px;
  color: var(--color-text);
  font-size: 13px;
}
.url-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.url-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 6px;
  margin-bottom: 4px;
  background: var(--color-bg);
  border: 1px solid transparent;
}
.url-list li.primary {
  border-color: var(--color-primary);
  background: rgba(102, 126, 234, 0.1);
}
.url-list li:last-child {
  margin-bottom: 0;
}
.url-list .url {
  flex: 1;
  font-size: 12px;
  word-break: break-all;
  color: var(--color-text);
}
.url-list .badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--color-primary);
  color: white;
  white-space: nowrap;
}
.firewall-section {
  margin-top: 16px;
  padding: 12px;
  background: rgba(220, 53, 69, 0.1);
  border-radius: 8px;
  border: 1px solid rgba(220, 53, 69, 0.3);
}
.firewall-section .sub-title {
  font-size: 12px;
  font-weight: 600;
  margin: 12px 0 4px;
  color: var(--color-text);
}
.firewall-section .sub-title:first-of-type {
  margin-top: 8px;
}
.firewall-section .section-title {
  font-weight: 600;
  margin: 0 0 8px;
  font-size: 13px;
}
.cmd-container {
  display: flex;
  gap: 8px;
  align-items: stretch;
  margin-top: 8px;
  padding: 6px 6px 6px 10px;
  background: #1e1e1e;
  border-radius: 6px;
}
.cmd-container .cmd {
  flex: 1;
  min-width: 0;
  overflow-x: auto;
  padding: 2px 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  white-space: nowrap;
  line-height: 24px;
}
.btn-copy {
  flex-shrink: 0;
  padding: 4px 12px;
  font-size: 12px;
  background: #0078d4;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}
.btn-copy:hover {
  background: #106ebe;
}

/* Path list (multiple Android save paths / loaded files) */
.path-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}
.path-item {
  display: flex;
  align-items: center;
  gap: 6px;
}
.path-item .path {
  flex: 1;
}
.badge-tag {
  flex-shrink: 0;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  background: #0078d4;
  color: white;
  white-space: nowrap;
}
.badge-tag.alt {
  background: #6c757d;
}
</style>
