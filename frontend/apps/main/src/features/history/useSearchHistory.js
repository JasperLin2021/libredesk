import { useStorage } from '@vueuse/core'
import { computed } from 'vue'

const STORAGE_KEY = 'libredesk.historySearchQueries'
const MAX_ENTRIES = 20
const MIN_LENGTH = 3

const normalize = (q) => (q || '').trim()

const isValid = (q) => normalize(q).length >= MIN_LENGTH

/**
 * Composable that persists the recent search history for the "历史 > 搜索"
 * page to localStorage. Entries are stored newest-first and deduplicated so
 * re-running a query just promotes it to the top of the list.
 */
export function useSearchHistory () {
  const records = useStorage(STORAGE_KEY, [], undefined, {
    serializer: {
      read: (raw) => {
        if (!raw) return []
        try {
          const parsed = JSON.parse(raw)
          return Array.isArray(parsed) ? parsed.filter((q) => typeof q === 'string') : []
        } catch (err) {
          console.error('Failed to parse history search records', err)
          return []
        }
      },
      write: (value) => JSON.stringify(value ?? [])
    }
  })

  const add = (query) => {
    const q = normalize(query)
    if (!isValid(q)) return
    const next = [q, ...records.value.filter((item) => item !== q)].slice(0, MAX_ENTRIES)
    records.value = next
  }

  const remove = (query) => {
    records.value = records.value.filter((item) => item !== query)
  }

  const clear = () => {
    records.value = []
  }

  const hasRecords = computed(() => records.value.length > 0)

  return { records, hasRecords, add, remove, clear, isValid }
}
