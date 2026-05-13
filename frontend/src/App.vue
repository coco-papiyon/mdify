<script lang="ts" setup>
import { ref } from 'vue'
import { ProcessClipboard } from '../wailsjs/go/main/App'

type ClipboardResult = {
  input: string
  output: string
  inputFormat: string
}

const formatOptions = [
  { value: 'AUTO', label: '自動判定' },
  { value: 'CF_UNICODETEXT', label: '通常テキスト (CF_UNICODETEXT)' },
  { value: 'CF_HTML', label: 'HTML 形式 (CF_HTML)' },
  { value: 'CF_RTF', label: 'RTF 形式 (CF_RTF)' },
] as const

const loading = ref(false)
const errorMessage = ref('')
const result = ref<ClipboardResult | null>(null)
const selectedFormat = ref<(typeof formatOptions)[number]['value']>('AUTO')

async function handleConvert() {
  loading.value = true
  errorMessage.value = ''

  try {
    result.value = await ProcessClipboard(selectedFormat.value)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    errorMessage.value = message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="app-shell">
    <section class="hero">
      <div class="hero-top">
        <div>
          <p class="eyebrow">Clipboard to Markdown</p>
          <h1>mdify</h1>
        </div>
        <div class="hero-actions">
          <label class="select-wrap">
            <span class="select-label">変換形式</span>
            <select v-model="selectedFormat" class="format-select">
              <option
                v-for="option in formatOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </option>
            </select>
          </label>
          <button class="convert-button" :disabled="loading" @click="handleConvert">
            {{ loading ? '変換中...' : 'クリップボードを変換' }}
          </button>
        </div>
      </div>
      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
    </section>

    <section class="panes">
      <article class="panel">
        <header class="panel-header">
          <div class="panel-title-row">
            <h2>変換前</h2>
            <span v-if="result" class="panel-format">{{ result.inputFormat }}</span>
          </div>
        </header>
        <pre class="panel-body">{{ result?.input || 'まだ変換していません。' }}</pre>
      </article>

      <article class="panel">
        <header class="panel-header">
          <h2>変換後</h2>
        </header>
        <pre class="panel-body">{{ result?.output || 'ボタンを押すとここに Markdown を表示します。' }}</pre>
      </article>
    </section>
  </main>
</template>
