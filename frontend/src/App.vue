<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'

import AndroidPanel from './components/AndroidPanel.vue'
import MigratePanel from './components/MigratePanel.vue'
import SlotTable from './components/SlotTable.vue'
import SteamPanel from './components/SteamPanel.vue'

import {
  AutoFetchAndroidDB,
  CheckWifiUpload,
  PickAndroidDBManually,
  ReadAndroidSlots,
  StartWifiServer,
  StopWifiServer,
} from '../bindings/wodima-slot-migrate/internal/android/service'
import { Detect, PickRemoteManually } from '../bindings/wodima-slot-migrate/internal/steam/service'
import { Migrate } from '../bindings/wodima-slot-migrate/internal/migrate/service'

import type { Result, SlotSelection } from '../bindings/wodima-slot-migrate/internal/migrate'
import type { Info } from '../bindings/wodima-slot-migrate/internal/steam'
import type { SlotRow } from '../bindings/wodima-slot-migrate/internal/android'

const steamInfo = ref<Info | null>(null)
const selectedRemote = ref('')
const dbPaths = ref<string[]>([])
const slots = ref<SlotRow[]>([])
const selectedKeys = ref<Set<string>>(new Set())
const results = ref<Result[]>([])

const busy = ref(false)
const steamError = ref('')
const androidError = ref('')
const androidStatus = ref('')

// Wi-Fi transfer state
const wifiUrl = ref('')
const wifiLocalUrl = ref('')
const wifiAllUrls = ref<string[]>([])
const wifiActive = ref(false)
const wifiWaiting = ref(false)
const wifiError = ref('')
const wifiStatus = ref('')
const wifiDebugInfo = ref('')
const wifiFirewallCmd = ref('')

const canMigrate = computed(
  () => selectedRemote.value !== '' && dbPaths.value.length > 0 && selectedKeys.value.size > 0,
)

// Build a stable composite key for a slot row so rows from different game.db
// files with the same SQLite rowid do not collide.
function slotKey(r: SlotRow): string {
  return r.sourceDb + '::' + r.id
}

// Map an on-device source path to a human-readable install type label.
function sourceLabel(sourcePath: string): string {
  if (sourcePath.includes('com.taptap')) return 'TapTap 启动'
  return '独立安装包'
}

onMounted(() => {
  void detectSteam()
})

async function detectSteam() {
  busy.value = true
  steamError.value = ''
  try {
    const resp = await Detect({})
    const info = resp?.info ?? null
    steamInfo.value = info
    if (info?.users && info.users.length === 1) {
      selectedRemote.value = info.users[0].remotePath
    }
  } catch (e: any) {
    steamError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

async function pickRemote() {
  busy.value = true
  steamError.value = ''
  try {
    const resp = await PickRemoteManually({})
    const dir = resp?.path ?? ''
    if (dir) selectedRemote.value = dir
  } catch (e: any) {
    steamError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

async function autoFetch() {
  await clearWifiState()
  clearLoadedFiles()
  busy.value = true
  androidError.value = ''
  androidStatus.value = '正在连接设备并拉取 game.db…'
  try {
    const resp = await AutoFetchAndroidDB({})
    const files = resp?.files ?? []
    if (files.length) {
      dbPaths.value = files.map((f) => f.path)
      androidStatus.value = `已获取 ${files.length} 个存档文件。`
      await readSlots(files)
    }
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
    androidStatus.value = ''
  } finally {
    busy.value = false
  }
}

async function pickAndroidDB() {
  await clearWifiState()
  clearLoadedFiles()
  busy.value = true
  androidError.value = ''
  androidStatus.value = ''
  try {
    const resp = await PickAndroidDBManually({})
    const paths = resp?.paths ?? []
    if (paths.length) {
      dbPaths.value = paths
      await readSlots(paths.map((p) => ({ path: p })))
    }
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

// readSlots reads slot rows from one or more game.db files. Each file may
// carry a sourcePath so rows can be labeled with the install type
// ("独立安装" / "TapTap 安装"). Without sourcePath the raw path is used.
async function readSlots(files: { path: string; sourcePath?: string }[]) {
  slots.value = []
  selectedKeys.value = new Set()
  results.value = []

  const allRows: SlotRow[] = []
  for (const f of files) {
    try {
      const resp = await ReadAndroidSlots({ dbPath: f.path })
      const rows = resp?.rows ?? []
      if (f.sourcePath) {
        for (const r of rows) r.sourceDb = sourceLabel(f.sourcePath)
      }
      allRows.push(...rows)
    } catch (e: any) {
      androidError.value = `读取 ${f.path} 失败：${String(e?.message ?? e)}`
    }
  }
  slots.value = allRows
}

function toggleSlot(key: string) {
  const next = new Set(selectedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  selectedKeys.value = next
}

function toggleAll(on: boolean) {
  selectedKeys.value = on ? new Set(slots.value.map(slotKey)) : new Set()
}

async function doMigrate() {
  busy.value = true
  results.value = []
  try {
    const selections: SlotSelection[] = slots.value
      .filter((r) => selectedKeys.value.has(slotKey(r)))
      .map((r) => ({
        id: r.id,
        slotIndex: r.slotIndex,
        jsonString: r.jsonString,
      }))

    const resp = await Migrate({ remotePath: selectedRemote.value, selections })
    results.value = resp?.results ?? []
  } catch (e: any) {
    androidError.value = String(e?.message ?? e)
  } finally {
    busy.value = false
  }
}

// Poll timer reference for cleanup
let wifiPollTimer: ReturnType<typeof setInterval> | null = null
let wifiPollTimeout: ReturnType<typeof setTimeout> | null = null

function stopWifiPolling() {
  if (wifiPollTimer) {
    clearInterval(wifiPollTimer)
    wifiPollTimer = null
  }
  if (wifiPollTimeout) {
    clearTimeout(wifiPollTimeout)
    wifiPollTimeout = null
  }
}

// Clear all Wi-Fi state and stop server if running
async function clearWifiState() {
  if (wifiActive.value) {
    stopWifiPolling()
    try {
      await StopWifiServer({})
    } catch {
      // Ignore errors when stopping
    }
  }
  wifiActive.value = false
  wifiUrl.value = ''
  wifiLocalUrl.value = ''
  wifiAllUrls.value = []
  wifiWaiting.value = false
  wifiError.value = ''
  wifiStatus.value = ''
  wifiDebugInfo.value = ''
  wifiFirewallCmd.value = ''
}

// Clear loaded file paths and slot rows so stale data does not linger when
// the user switches acquisition methods.
function clearLoadedFiles() {
  dbPaths.value = []
  slots.value = []
  selectedKeys.value = new Set()
  results.value = []
  androidStatus.value = ''
  androidError.value = ''
}

async function toggleWifi() {
  if (wifiActive.value) {
    // Stop Wi-Fi server
    stopWifiPolling()
    try {
      await StopWifiServer({})
    } catch (e: any) {
      wifiError.value = String(e?.message ?? e)
    }
    wifiActive.value = false
    wifiUrl.value = ''
    wifiLocalUrl.value = ''
    wifiAllUrls.value = []
    wifiWaiting.value = false
    wifiError.value = ''
    wifiDebugInfo.value = ''
    wifiFirewallCmd.value = ''
  } else {
    // Start Wi-Fi server
    clearLoadedFiles()
    busy.value = true
    wifiError.value = ''
    wifiStatus.value = '正在启动 Wi-Fi 服务器…'
    try {
      const resp = await StartWifiServer({})
      const result = resp?.result ?? null
      if (!result) {
        wifiError.value = '启动 Wi-Fi 服务器失败：返回结果为空'
        wifiStatus.value = ''
        return
      }

      wifiUrl.value = result.url
      wifiLocalUrl.value = result.localUrl
      wifiAllUrls.value = result.allUrls || []
      wifiDebugInfo.value = result.debugInfo
      wifiFirewallCmd.value = result.firewallCmd || ''
      wifiActive.value = true
      wifiWaiting.value = true
      wifiStatus.value = '等待手机上传存档文件…'
      // Poll for upload completion
      pollWifiUpload(result.token)
    } catch (e: any) {
      wifiError.value = String(e?.message ?? e)
      wifiStatus.value = ''
    } finally {
      busy.value = false
    }
  }
}

function pollWifiUpload(token: string) {
  // Clear any existing polling
  stopWifiPolling()

  // Poll every 2 seconds for file upload completion
  wifiPollTimer = setInterval(async () => {
    try {
      const resp = await CheckWifiUpload({ token })
      const path = resp?.path ?? ''
      if (path) {
        // File uploaded successfully
        stopWifiPolling()
        wifiWaiting.value = false
        wifiActive.value = false
        wifiUrl.value = ''
        wifiLocalUrl.value = ''
        wifiStatus.value = ''
        dbPaths.value = [path]
        androidStatus.value = 'Wi-Fi 上传成功，已加载存档文件。'
        await readSlots([{ path }])
      }
    } catch {
      // Still waiting, ignore errors
    }
  }, 2000)

  // Timeout after 10 minutes
  wifiPollTimeout = setTimeout(
    () => {
      stopWifiPolling()
      if (wifiActive.value && wifiWaiting.value) {
        wifiWaiting.value = false
        wifiStatus.value = '等待超时，请重新启动 Wi-Fi 传输。'
        wifiError.value = '上传超时（10 分钟）'
      }
    },
    10 * 60 * 1000,
  )
}
</script>

<template>
  <div class="app">
    <header class="app-header">
      <h1>我在地府打麻将 · 存档迁移</h1>
      <p class="subtitle">Android → Steam (appid 3444020)</p>
    </header>
    <main class="app-main">
      <SteamPanel
        :info="steamInfo"
        :selected-remote="selectedRemote"
        :busy="busy"
        :error="steamError"
        @detect="detectSteam"
        @pick="pickRemote"
        @select="(r: string) => (selectedRemote = r)"
      />
      <AndroidPanel
        :db-paths="dbPaths"
        :busy="busy"
        :status="androidStatus"
        :error="androidError"
        :wifi-url="wifiUrl"
        :wifi-local-url="wifiLocalUrl"
        :wifi-all-urls="wifiAllUrls"
        :wifi-active="wifiActive"
        :wifi-waiting="wifiWaiting"
        :wifi-error="wifiError"
        :wifi-status="wifiStatus"
        :wifi-debug-info="wifiDebugInfo"
        :wifi-firewall-cmd="wifiFirewallCmd"
        @auto="autoFetch"
        @pick="pickAndroidDB"
        @wifi="toggleWifi"
      />
      <SlotTable
        :rows="slots"
        :selected-keys="selectedKeys"
        :busy="busy"
        @toggle="toggleSlot"
        @toggle-all="toggleAll"
      />
      <MigratePanel
        :results="results"
        :busy="busy"
        :can-migrate="canMigrate"
        @migrate="doMigrate"
      />
    </main>
  </div>
</template>
