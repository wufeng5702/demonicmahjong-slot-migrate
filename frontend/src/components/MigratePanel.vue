<script lang="ts" setup>
import { computed } from 'vue'
import type { Result } from '../../bindings/demonicmahjong-slot-migrate/internal/migrate'

const props = defineProps<{
  results: Result[]
  busy: boolean
  canMigrate: boolean
}>()

const emit = defineEmits<{
  (_e: 'migrate'): void
}>()

const summary = computed(() => {
  const ok = props.results.filter((r) => r.success).length
  const fail = props.results.length - ok
  return { ok, fail, total: props.results.length }
})
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>4. 迁移</h2>
    </header>
    <div class="panel-body">
      <div class="row">
        <button
          class="btn btn-primary"
          :disabled="props.busy || !props.canMigrate"
          @click="emit('migrate')"
        >
          开始迁移
        </button>
        <span v-if="props.results.length" class="hint">
          成功 {{ summary.ok }} / 失败 {{ summary.fail }} / 共 {{ summary.total }}
        </span>
      </div>
      <ul v-if="props.results.length" class="result-log">
        <li v-for="r in props.results" :key="r.id" :class="r.success ? 'ok' : 'fail'">
          <span class="badge">{{ r.success ? 'OK' : 'FAIL' }}</span>
          Slot{{ r.slotIndex }} →
          <code>{{ r.targetFile }}</code>
          <span v-if="r.backupFile" class="backup">(已备份: {{ r.backupFile }})</span>
          <span v-if="r.error" class="error">{{ r.error }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>
