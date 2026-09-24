import { computed, ref, watch, type Ref } from 'vue'

export const PAGE_SIZES = [10, 20, 50, 100]

export function usePagination<T>(source: Ref<T[]>) {
  const page = ref(1)
  const pageSize = ref(10)

  const total = computed(() => source.value.length)
  const paged = computed(() => {
    const start = (page.value - 1) * pageSize.value
    return source.value.slice(start, start + pageSize.value)
  })

  watch([total, pageSize], () => {
    const maxPage = Math.max(1, Math.ceil(total.value / pageSize.value) || 1)
    if (page.value > maxPage) page.value = maxPage
  })

  function resetPage() {
    page.value = 1
  }

  return { page, pageSize, total, paged, pageSizes: PAGE_SIZES, resetPage }
}
