<script lang="ts" setup>
import { computed } from 'vue'
import type { Info } from '../../bindings/demonicmahjong-slot-migrate/internal/steam'

const props = defineProps<{
  info: Info | null
  selectedRemote: string
  busy: boolean
  error: string
}>()

const emit = defineEmits<{
  (_e: 'detect'): void
  (_e: 'pick'): void
  (_e: 'select', _remote: string): void
}>()

const users = computed(() => props.info?.users ?? [])

function onPick() {
  emit('pick')
}
</script>

<template>
  <section class="panel">
    <header class="panel-header">
      <h2>1. Steam 存档目录</h2>
      <button class="btn btn-sm" :disabled="props.busy" @click="emit('detect')">重新检测</button>
    </header>
    <div v-if="props.info" class="panel-body">
      <div class="row">
        <span class="label">Steam 安装目录：</span>
        <code class="path">{{ props.info.steamPath }}</code>
      </div>
      <div v-if="users.length" class="row">
        <span class="label">检测到的用户存档：</span>
        <select
          class="select"
          :value="props.selectedRemote"
          :disabled="props.busy"
          @change="emit('select', ($event.target as HTMLSelectElement).value)"
        >
          <option value="">— 请选择 —</option>
          <option v-for="u in users" :key="u.steamId" :value="u.remotePath">
            steamID {{ u.steamId }}
          </option>
        </select>
      </div>
      <div v-else class="row">
        <span class="hint">未找到拥有该游戏存档的 Steam 用户。可手动选择目录：</span>
      </div>
      <div class="row">
        <button class="btn btn-sm" :disabled="props.busy" @click="onPick">
          手动选择 remote 目录
        </button>
        <code
          v-if="props.selectedRemote && !users.some((u) => u.remotePath === props.selectedRemote)"
          class="path"
        >
          {{ props.selectedRemote }}
        </code>
      </div>
    </div>
    <div v-else class="panel-body">
      <p v-if="props.error" class="error">
        {{ props.error }}
      </p>
      <p v-else class="hint">正在检测…</p>
    </div>
  </section>
</template>
