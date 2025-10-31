<template>
  <transition name="flow-fade" mode="out-in">
    <component :is="activeStep.component" :key="activeStep.id" />
  </transition>
</template>

<script setup>
import { computed, provide, ref, watch } from 'vue'
import StepOne from '../components/flow/StepOne.vue'
import StepTwo from '../components/flow/StepTwo.vue'
import StepThree from '../components/flow/StepThree.vue'
import StepFour from '../components/flow/StepFour.vue'
import StepFive from '../components/flow/StepFive.vue'
import { useFlowStore } from '../stores/flow'

const flowStore = useFlowStore()

const steps = [
  { id: 1, component: StepOne },
  { id: 2, component: StepTwo },
  { id: 3, component: StepThree },
  { id: 4, component: StepFour },
  { id: 5, component: StepFive },
]

const activeIndex = ref(Math.max(0, flowStore.step - 1))

const unlockIndex = computed(() => Math.max(flowStore.step - 1, 0))

const activeStep = computed(() => steps[Math.min(activeIndex.value, steps.length - 1)])

const goPrev = () => {
  if (activeIndex.value === 0) return
  activeIndex.value -= 1
  syncStep()
}

const goNext = () => {
  if (activeIndex.value >= steps.length - 1) return
  if (activeIndex.value >= unlockIndex.value) return
  activeIndex.value += 1
  syncStep()
}

const goTo = (index) => {
  if (index < 0 || index >= steps.length) return
  if (index > unlockIndex.value) return
  activeIndex.value = index
  syncStep()
}

const syncStep = () => {
  const current = activeIndex.value + 1
  if (current > flowStore.step) {
    flowStore.step = current
  }
}

watch(
  () => flowStore.step,
  (value) => {
    const target = Math.max(0, Math.min(value - 1, steps.length - 1))
    if (target !== activeIndex.value) {
      activeIndex.value = target
    }
  }
)

provide('flowNav', {
  goPrev,
  goNext,
  goTo,
  activeIndex: computed(() => activeIndex.value),
  unlockIndex,
})
</script>

<style scoped>
.flow-fade-enter-active,
.flow-fade-leave-active {
  transition: opacity 0.2s ease;
}

.flow-fade-enter-from,
.flow-fade-leave-to {
  opacity: 0;
}
</style>
