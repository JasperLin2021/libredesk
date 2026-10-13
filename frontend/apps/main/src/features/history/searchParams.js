/**
 * Builds the query parameter object passed to `api.searchConversations` and
 * `api.searchMessages`. The keys mirror what the Go backend expects in
 * `cmd/search.go` (`parseSearchFilters`). Unset filters are stripped so the
 * backend receives the same request shape as before and the prepared
 * statement's `IS NULL` skip-pattern can apply.
 *
 * @param {object} args
 * @param {string} args.query
 * @param {object} args.filters
 * @param {string} [args.filters.sort]
 * @param {string} [args.filters.period]
 * @param {string} [args.filters.start]
 * @param {string} [args.filters.end]
 * @param {number[]} [args.filters.inboxIds]
 * @param {number} [args.filters.agentId]
 */
export function buildSearchParams ({ query, filters = {} }) {
  const params = {}
  const q = (query || '').trim()
  if (q) params.query = q

  if (filters.sort) params.sort = filters.sort
  if (filters.period) params.period = filters.period
  if (filters.start) params.start = filters.start
  if (filters.end) params.end = filters.end
  if (Array.isArray(filters.inboxIds) && filters.inboxIds.length > 0) {
    params.inbox_ids = filters.inboxIds.join(',')
  }
  if (filters.agentId && filters.agentId > 0) {
    params.agent_id = filters.agentId
  }
  return params
}

/** Human-readable period options used by both the dropdown and the helper. */
export const PERIOD_OPTIONS = [
  { value: 'today', labelKey: 'history.filter.period.today' },
  { value: 'yesterday', labelKey: 'history.filter.period.yesterday' },
  { value: 'this_week', labelKey: 'history.filter.period.thisWeek' },
  { value: 'this_month', labelKey: 'history.filter.period.thisMonth' },
  { value: 'last_month', labelKey: 'history.filter.period.lastMonth' },
  { value: 'custom', labelKey: 'history.filter.period.custom' }
]

/** Allowed sort orderings for the time-sort segmented buttons. */
export const SORT_OPTIONS = [
  { value: 'newest', labelKey: 'history.filter.sort.newest' },
  { value: 'oldest', labelKey: 'history.filter.sort.oldest' }
]

/** Sentinel value used in the UI to mean "all agents". */
export const ALL_AGENTS = 0
