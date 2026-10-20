// Shared option definitions for the keyword match (关键字匹配) rules. The values
// mirror the backend enums in internal/keywordmatch/models.
export const KEYWORD_MATCH_SOURCES = [
  { value: 'visitor', labelKey: 'keywordMatch.source.visitor' },
  { value: 'agent', labelKey: 'keywordMatch.source.agent' }
]

export const KEYWORD_MATCH_TYPES = [
  { value: 'keyword', labelKey: 'keywordMatch.matchType.keyword' },
  { value: 'regex', labelKey: 'keywordMatch.matchType.regex' }
]

export const KEYWORD_MATCH_ACTIONS = [
  { value: 'agent_notify', labelKey: 'keywordMatch.action.agentNotify' },
  { value: 'agent_and_visitor', labelKey: 'keywordMatch.action.agentAndVisitor' }
]

// emptyRule returns a blank rule pre-filled with the backend defaults.
export function emptyRule () {
  return {
    name: '',
    source: 'visitor',
    match_type: 'keyword',
    pattern: '',
    case_sensitive: false,
    search_first: false,
    action: 'agent_notify',
    reply_content: '',
    enabled: true,
    sort_order: 0
  }
}

// toForm converts an API rule into the dialog form shape.
export function toForm (rule) {
  return { ...emptyRule(), ...(rule || {}) }
}

function labelOf (options, value, t) {
  const found = options.find((option) => option.value === value)
  return found ? t(found.labelKey) : value || '-'
}

export const sourceLabel = (value, t) => labelOf(KEYWORD_MATCH_SOURCES, value, t)
export const matchTypeLabel = (value, t) => labelOf(KEYWORD_MATCH_TYPES, value, t)
export const actionLabel = (value, t) => labelOf(KEYWORD_MATCH_ACTIONS, value, t)

// isValidRegex reports whether the pattern is a usable JavaScript regex.
export function isValidRegex (pattern) {
  try {
    new RegExp(pattern)
    return true
  } catch {
    return false
  }
}
