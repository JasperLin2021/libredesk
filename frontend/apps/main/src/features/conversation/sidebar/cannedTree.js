// Shared tree helpers for the canned responses sidebar. Both the panel and
// the recursive category group need to prune the global category tree down
// to the branches containing search matches, so the logic lives here once.

// `matchCategory` returns a copy of `cat` containing only the responses (and
// nested sub-categories) whose content matches `q` (case-insensitive
// substring). Returns null when nothing in the subtree matches, so callers
// can drop the branch entirely.
export const matchCategory = (cat, q) => {
  const responses = (cat.responses || []).filter((r) =>
    (r.content || '').toLowerCase().includes(q)
  )
  const children = (cat.children || [])
    .map((child) => matchCategory(child, q))
    .filter(Boolean)
  if (responses.length === 0 && children.length === 0) return null
  return { ...cat, responses, children }
}

// `filterCannedTree` prunes a whole category forest to matching branches.
// With an empty query the original tree is returned untouched.
export const filterCannedTree = (tree, query) => {
  const q = (query || '').trim().toLowerCase()
  if (!q) return tree || []
  return (tree || []).map((cat) => matchCategory(cat, q)).filter(Boolean)
}
