<template>
  <FlowLayout :step="0" :total="0" title="客户列表" subtitle="集中查看并维护所有客户数据。">
    <section class="customers">
      <header class="customers__toolbar">
        <label class="customers__filter">
          <span>评级</span>
          <select v-model="selectedFilters.grade" @change="onSelectChange('grade', selectedFilters.grade)">
            <option value="">全部</option>
            <option v-for="grade in gradeOptions" :key="grade" :value="grade">{{ grade }}</option>
          </select>
        </label>
        <label class="customers__filter">
          <span>国家/地区</span>
          <select v-model="selectedFilters.country" @change="onSelectChange('country', selectedFilters.country)">
            <option value="">全部</option>
            <option v-for="country in countryOptions" :key="country" :value="country">{{ country }}</option>
          </select>
        </label>
        <label class="customers__filter">
          <span>跟进状态</span>
          <select v-model="selectedFilters.status" @change="onSelectChange('status', selectedFilters.status)">
            <option value="">全部</option>
            <option value="pending">待跟进</option>
            <option value="in_progress">跟进中</option>
            <option value="won">已成交</option>
          </select>
        </label>
        <label class="customers__filter">
          <span>排序</span>
          <select v-model="selectedFilters.sort" @change="onSelectChange('sort', selectedFilters.sort)">
            <option value="last_followup_desc">按最近跟进</option>
            <option value="last_followup_asc">按最早跟进</option>
            <option value="name_asc">按公司名 A → Z</option>
            <option value="name_desc">按公司名 Z → A</option>
          </select>
        </label>
      </header>

      <div class="customers__table">
        <table>
          <thead>
            <tr>
              <th>公司</th>
              <th>国家/地区</th>
              <th>评级</th>
              <th>最后跟进</th>
              <th class="actions">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="empty">加载中…</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="5" class="empty">暂无客户记录</td>
            </tr>
            <tr v-for="customer in items" :key="customer.id">
              <td class="name">
                <p class="title">{{ customer.name }}</p>
                <p class="hint">最近更新时间：{{ formatDisplayDate(customer.updated_at) }}</p>
              </td>
              <td>{{ customer.country || '—' }}</td>
              <td>
                <span :class="['badge', `badge--${(customer.grade || 'unknown').toLowerCase()}`]">{{ customer.grade }}</span>
              </td>
              <td>{{ formatDisplayDate(customer.last_followup_at) }}</td>
              <td class="actions">
                <button type="button" @click="openEditor(customer.id)">编辑</button>
                <button type="button" class="danger" @click="confirmDelete(customer)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <CustomerEditModal
      v-if="isEditorVisible"
      :customer="detail"
      :loading="detailLoading"
      @close="closeEditor"
      @updated="onCustomerUpdated"
    />
  </FlowLayout>
</template>

<script setup>
import { computed, reactive, onMounted, watch, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import FlowLayout from '../components/flow/FlowLayout.vue'
import CustomerEditModal from '../components/customers/CustomerEditModal.vue'
import { useCustomersStore } from '../stores/customers'

const customersStore = useCustomersStore()
const { items, loading, detail, detailLoading, filters } = storeToRefs(customersStore)

const selectedFilters = reactive({
  grade: filters.value.grade,
  country: filters.value.country,
  status: filters.value.status,
  sort: filters.value.sort,
})

const gradeOptions = computed(() => {
  const set = new Set(['S', 'A', 'B', 'C'])
  items.value.forEach((item) => {
    if (item.grade) set.add(item.grade)
  })
  return Array.from(set)
})

const countryOptions = computed(() => {
  const set = new Set()
  items.value.forEach((item) => {
    if (item.country) set.add(item.country)
  })
  return Array.from(set).sort()
})

const isEditorVisible = computed(() => detailLoading.value || !!detail.value)

const refreshList = () => {
  customersStore.fetchList()
}

const onSelectChange = (key, value) => {
  customersStore.setFilter(key, value)
  refreshList()
}

watch(filters, (value) => {
  selectedFilters.grade = value.grade
  selectedFilters.country = value.country
  selectedFilters.status = value.status
  selectedFilters.sort = value.sort
})

const formatDisplayDate = (value) => {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString()
}

const openEditor = async (customerId) => {
  await customersStore.fetchDetail(customerId)
}

const closeEditor = () => {
  customersStore.clearDetail()
}

const onCustomerUpdated = () => {
  closeEditor()
  refreshList()
}

const confirmDelete = async (customer) => {
  if (!customer || !customer.id) return
  const name = customer.name ? `「${customer.name}」` : '该客户'
  const confirmed = window.confirm(`确定要删除${name}吗？此操作不可撤销。`)
  if (!confirmed) return
  await customersStore.removeCustomer(customer.id)
}

onMounted(() => {
  customersStore.fetchList()
})

onUnmounted(() => {
  // no-op, placeholder for consistency
})
</script>

<style scoped>
.customers {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.customers__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: center;
}

.customers__filter {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.customers__filter select {
  min-width: 140px;
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px solid var(--border-default);
  background: #fff;
  font-size: 14px;
}

.customers__table {
  background: var(--surface-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-default);
  box-shadow: var(--shadow-card);
  overflow: hidden;
}

.customers__table table {
  width: 100%;
  border-collapse: collapse;
}

.customers__table th {
  text-align: left;
  padding: 14px 20px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.customers__table td {
  padding: 16px 20px;
  border-top: 1px solid var(--border-default);
  font-size: 14px;
  color: var(--text-primary);
  vertical-align: middle;
}

.customers__table td.name .title {
  margin: 0;
  font-weight: 600;
}

.customers__table td.name .hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--text-tertiary);
}

.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 24px;
  padding: 0 10px;
  font-size: 12px;
  border-radius: 12px;
  font-weight: 600;
}

.badge--s {
  background: rgba(234, 179, 8, 0.18);
  color: #b45309;
}

.badge--a {
  background: rgba(59, 130, 246, 0.18);
  color: #1d4ed8;
}

.badge--b {
  background: rgba(99, 102, 241, 0.18);
  color: #4338ca;
}

.badge--c {
  background: rgba(148, 163, 184, 0.2);
  color: #475569;
}

.badge--unknown {
  background: rgba(148, 163, 184, 0.12);
  color: #64748b;
}

.customers__table td.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.customers__table td.actions button {
  border: 1px solid var(--border-default);
  background: #fff;
  border-radius: 10px;
  padding: 6px 16px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.customers__table td.actions button:hover {
  background: var(--surface-subtle);
}

.customers__table td.actions button.danger {
  border-color: rgba(239, 68, 68, 0.4);
  color: #b91c1c;
}

.customers__table td.actions button.danger:hover {
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.7);
}

.empty {
  text-align: center;
  padding: 40px 12px;
  color: var(--text-tertiary);
}

@media (max-width: 768px) {
  .customers__toolbar {
    gap: 12px;
  }

  .customers__filter {
    flex: 1;
  }

  .customers__filter select {
    width: 100%;
  }
}
</style>
