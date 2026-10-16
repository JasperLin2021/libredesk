/**
 * Query parameter builders and option lists shared by the two "客户" pages
 * (客户信息 / 操作日志). The keys mirror the Go handlers in
 * `cmd/blacklist.go`.
 */

/** Period options of the blacklist filter panels. */
export const BLACKLIST_PERIOD_OPTIONS = [
  { value: 'all', labelKey: 'history.filter.periodPlaceholder' },
  { value: 'this_week', labelKey: 'history.filter.period.thisWeek' },
  { value: 'today', labelKey: 'history.filter.period.today' },
  { value: 'yesterday', labelKey: 'history.filter.period.yesterday' },
  { value: 'this_month', labelKey: 'history.filter.period.thisMonth' },
  { value: 'last_month', labelKey: 'history.filter.period.lastMonth' },
  { value: 'custom', labelKey: 'history.filter.period.custom' }
]

/** Sort orderings of the blacklist filter panels. */
export const BLACKLIST_SORT_OPTIONS = [
  { value: 'newest', labelKey: 'history.filter.sort.newest' },
  { value: 'oldest', labelKey: 'history.filter.sort.oldest' }
]

/** Operation type filter of the operation log page. */
export const BLACKLIST_ACTION_OPTIONS = [
  { value: '', labelKey: 'history.logs.action.all' },
  { value: 'add', labelKey: 'history.logs.action.add' },
  { value: 'remove', labelKey: 'history.logs.action.remove' }
]

/** Default filter set of both pages. */
export function defaultBlacklistFilters () {
  return {
    sort: 'newest',
    period: 'all',
    start: '',
    end: '',
    search: '',
    action: ''
  }
}

/**
 * Builds the params for `api.getBlacklistCustomers`.
 *
 * @param {object} args
 * @param {object} args.filters
 * @param {string} args.tab – 'all' | 'blocked'
 * @param {number} args.page
 * @param {number} args.pageSize
 */
export function buildBlacklistCustomersParams ({ filters = {}, tab = 'all', page = 1, pageSize = 30 } = {}) {
  return {
    ...commonParams(filters, page, pageSize),
    filter: tab === 'blocked' ? 'blocked' : 'all'
  }
}

/**
 * Builds the params for `api.getBlacklistLogs`.
 */
export function buildBlacklistLogsParams ({ filters = {}, page = 1, pageSize = 30 } = {}) {
  const params = commonParams(filters, page, pageSize)
  if (filters.action) params.action = filters.action
  return params
}

function commonParams (filters, page, pageSize) {
  const params = {
    page,
    page_size: pageSize,
    order: filters.sort === 'oldest' ? 'asc' : 'desc'
  }
  if (filters.period && filters.period !== 'all') {
    params.period = filters.period
    if (filters.period === 'custom') {
      if (filters.start) params.start = filters.start
      if (filters.end) params.end = filters.end
    }
  }
  const search = (filters.search || '').trim()
  if (search) params.search = search
  return params
}
