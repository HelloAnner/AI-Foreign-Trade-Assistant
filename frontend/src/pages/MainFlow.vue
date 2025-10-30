<template>
  <div class="layout">
    <header class="topbar">
      <div class="title">AI 外贸客户开发助手</div>
      <button class="settings-btn" @click="goSettings">设置</button>
    </header>

    <div class="carousel">
      <button class="nav-btn" :disabled="activeIndex === 0" @click="goPrev">‹</button>
      <div class="viewport">
        <transition :name="transitionName" mode="out-in">
          <component :is="steps[activeIndex].component" :key="steps[activeIndex].id" />
        </transition>
      </div>
      <button class="nav-btn" :disabled="!canGoNext" @click="goNext">›</button>
    </div>

    <footer class="indicator">
      <span>Step {{ activeIndex + 1 }} / {{ steps.length }}</span>
      <div class="dots">
        <button
          v-for="(step, index) in steps"
          :key="step.id"
          :class="['dot', { 'dot--active': index === activeIndex, 'dot--unlock': index < unlockIndex }]"
          :disabled="index > unlockIndex"
          @click="goTo(index)"
        ></button>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { computed, ref, watch, provide } from 'vue'
import { useRouter } from 'vue-router'
import StepOne from '../components/flow/StepOne.vue'
import StepTwo from '../components/flow/StepTwo.vue'
import StepThree from '../components/flow/StepThree.vue'
import StepFour from '../components/flow/StepFour.vue'
import StepFive from '../components/flow/StepFive.vue'
import { useFlowStore } from '../stores/flow'

const router = useRouter()
const flowStore = useFlowStore()

const steps = [
  { id: 1, component: StepOne },
  { id: 2, component: StepTwo },
  { id: 3, component: StepThree },
  { id: 4, component: StepFour },
  { id: 5, component: StepFive },
]

const activeIndex = ref(0)
const direction = ref('forward')

const unlockIndex = computed(() => Math.max(flowStore.step - 1, 0))
const canGoNext = computed(() => activeIndex.value < unlockIndex.value)
const transitionName = computed(() => (direction.value === 'forward' ? 'slide-left' : 'slide-right'))

const nav = {
  goPrev,
  goNext,
  goTo,
}

provide('flowNav', nav)

watch(
  () => flowStore.step,
  (value) => {
    const target = Math.max(0, value - 1)
    if (target !== activeIndex.value) {
      direction.value = target > activeIndex.value ? 'forward' : 'backward'
      activeIndex.value = target
    }
  }
)

function goPrev() {
  if (activeIndex.value === 0) return
  direction.value = 'backward'
  activeIndex.value -= 1
  syncStoreStep()
}

function goNext() {
  if (!canGoNext.value) return
  direction.value = 'forward'
  activeIndex.value += 1
  syncStoreStep()
}

function goTo(index) {
  if (index > unlockIndex.value) return
  direction.value = index > activeIndex.value ? 'forward' : 'backward'
  activeIndex.value = index
  syncStoreStep()
}

function syncStoreStep() {
  const current = activeIndex.value + 1
  if (current > flowStore.step) {
    flowStore.step = current
  }
}

const goSettings = () => {
  router.push({ name: 'settings' })
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--surface-background);
  color: var(--text-primary);
  padding: 32px 48px;
  gap: 24px;
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title {
  font-size: 28px;
  font-weight: 700;
}

.settings-btn {
  border: 1px solid var(--border-subtle);
  border-radius: 999px;
  background: transparent;
  padding: 10px 24px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.settings-btn:hover {
  border-color: var(--accent-color);
  color: var(--accent-color);
}

.carousel {
  flex: 1;
  display: flex;
  align-items: stretch;
  gap: 16px;
}

.nav-btn {
  width: 48px;
  border: none;
  border-radius: 18px;
  background: rgba(148, 163, 184, 0.18);
  color: var(--text-secondary);
  font-size: 28px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.viewport {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.35s ease, opacity 0.35s ease;
  position: absolute;
  inset: 0;
}

.slide-left-enter-from {
  transform: translateX(100%);
  opacity: 0.4;
}

.slide-left-leave-to {
  transform: translateX(-100%);
  opacity: 0.2;
}

.slide-right-enter-from {
  transform: translateX(-100%);
  opacity: 0.4;
}

.slide-right-leave-to {
  transform: translateX(100%);
  opacity: 0.2;
}

.viewport > * {
  position: absolute;
  inset: 0;
}

.indicator {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--text-tertiary);
}

.dots {
  display: flex;
  gap: 8px;
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: none;
  background: var(--border-subtle);
  cursor: pointer;
  transition: background 0.2s ease;
}

.dot:disabled {
  cursor: not-allowed;
}

.dot--unlock {
  background: rgba(37, 99, 235, 0.25);
}

.dot--active {
  background: var(--accent-color);
}
</style>
