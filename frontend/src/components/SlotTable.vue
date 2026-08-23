<script lang="ts" setup>
import { computed } from 'vue'
import type { SlotRow } from '../../bindings/wodima-slot-migrate/internal/android'

const props = defineProps<{
  rows: SlotRow[]
  selectedKeys: Set<string>
  busy: boolean
}>()

const emit = defineEmits<{
  (_e: 'toggle', _key: string): void
  (_e: 'toggleAll', _on: boolean): void
}>()

// Build a stable composite key matching App.vue's slotKey().
function slotKey(r: SlotRow): string {
  return r.sourceDb + '::' + r.id
}

const allSelected = computed(
  () => props.rows.length > 0 && props.rows.every((r) => props.selectedKeys.has(slotKey(r))),
)

function formatSize(n: number): string {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(2) + ' MB'
}

// Extract a short label from the sourceDb value. For auto-fetched rows this
// is already a label like "独立安装" or "TapTap 安装". For manually picked
// files it's a full path, so we fall back to the filename.
function shortDb(sourceDb: string): string {
  if (!sourceDb) return ''
  // If it doesn't look like a path (no / or \), it's already a label.
  if (!sourceDb.includes('/') && !sourceDb.includes('\\')) return sourceDb
  const parts = sourceDb.replace(/\\/g, '/').split('/')
  return parts[parts.length - 1] || sourceDb
}

// Detect slotIndex collisions across selected rows so we can warn the user:
// migrating two rows into the same Slot{X}.json would overwrite each other.
const slotCollision = computed(() => {
  const seen = new Map<number, string>()
  for (const r of props.rows) {
    if (props.selectedKeys.has(slotKey(r))) {
      if (seen.has(r.slotIndex)) return r.slotIndex
      seen.set(r.slotIndex, slotKey(r))
    }
  }
  return -1
})

// Whether more than one distinct source db is present in the table.
const hasMultipleSources = computed(() => {
  const seen = new Set<string>()
  for (const r of props.rows) {
    seen.add(r.sourceDb)
    if (seen.size > 1) return true
  }
  return false
})
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>3. 选择要迁移的存档</h2>
      <label v-if="props.rows.length" class="check-all">
        <input
          type="checkbox"
          :checked="allSelected"
          :disabled="props.busy"
          @change="emit('toggleAll', !allSelected)"
        />
        全选
      </label>
    </header>
    <div class="panel-body">
      <p v-if="!props.rows.length" class="hint">请先选择安卓存档文件。</p>
      <p v-else-if="slotCollision >= 0" class="warn">
        注意：选中的行中有多条对应 Slot{{ slotCollision }}，迁移时后写入的会覆盖先写入的。
      </p>
      <table v-if="props.rows.length" class="slot-table">
        <thead>
          <tr>
            <th class="col-check" />
            <th class="col-slot">slotIndex</th>
            <th class="col-account">userAccount</th>
            <th v-if="hasMultipleSources" class="col-source">来源</th>
            <th class="col-size">JSON 大小</th>
            <th class="col-preview">JSON 预览</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in props.rows" :key="slotKey(r)">
            <td class="col-check">
              <input
                type="checkbox"
                :checked="props.selectedKeys.has(slotKey(r))"
                :disabled="props.busy"
                @change="emit('toggle', slotKey(r))"
              />
            </td>
            <td class="col-slot">
              {{ r.slotIndex }}
            </td>
            <td class="col-account">
              {{ r.userAccount || '(空)' }}
            </td>
            <td v-if="hasMultipleSources" class="col-source" :title="r.sourceDb">
              {{ shortDb(r.sourceDb) }}
            </td>
            <td class="col-size">
              {{ formatSize(r.jsonSize) }}
            </td>
            <td class="col-preview">
              <code>{{ r.jsonPreview }}</code>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.col-source {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  color: #888;
}
</style>
