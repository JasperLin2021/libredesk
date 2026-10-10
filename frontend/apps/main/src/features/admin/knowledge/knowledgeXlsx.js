import * as XLSX from 'xlsx'

// Shared helpers for the Excel template / import / export of knowledge
// articles. The template uses seven columns:
//   A: category code (分类Code, the level-1 "知识库" identifier used by the
//      importer to look up or auto-create the parent category)
//   B: category name (分类名 — the level-2 "分类" the article belongs to)
//   C: external id (知识ID, the upsert key: existing rows are updated, new
//      rows are inserted)
//   D: article code (知识Code, optional third-party identifier)
//   E: title (知识标题, required)
//   F: author (知识作者, optional free-form text)
//   G: content (知识内容, rich text body)
//
// The first row is reserved for column headers; the second row carries the
// per-column hints so the user can edit the file in place.
//
// Each column carries an inline fallback label/hint used when the i18n pack
// does not define the key (the backend snapshots i18n files at boot, so a
// freshly added key only shows up after a restart).
const HEADERS = [
  { key: 'admin.knowledgeBase.template.categoryCode', label: '分类Code', hint: '分类Code 必填' },
  { key: 'admin.knowledgeBase.template.categoryName', label: '分类名', hint: "一级/二级 分类名，例如：其他（通用）知识库/游戏通用话术" },
  { key: 'admin.knowledgeBase.template.externalId', label: '知识ID', hint: '已有可更新，不存在时新添' },
  { key: 'admin.knowledgeBase.template.articleCode', label: '知识Code', hint: '第三方知识标识，没有可不设置' },
  { key: 'admin.knowledgeBase.template.title', label: '知识标题', hint: '50个字，超过则无法正常导入' },
  { key: 'admin.knowledgeBase.template.author', label: '知识作者', hint: '允许输入文本格式内容，不限制大小' },
  { key: 'admin.knowledgeBase.template.content', label: '知识内容', hint: '支持富文本(HTML)，可包含图片与链接' }
]

const cellText = (value) => String(value ?? '').trim()

// Translates `key`, falling back to the built-in default when the i18n pack
// has no entry (vue-i18n returns the key itself in that case).
const tr = (t, key, fallback) => {
  const v = t(key)
  return !v || v === key ? fallback : v
}

const headerRow = (t) => HEADERS.map((h) => tr(t, h.key, h.label))

const isHeaderRow = (row) => {
  if (!row) return false
  const joined = row.map((c) => cellText(c).toLowerCase()).join(' ')
  return joined.includes('分类') || joined.includes('category')
}

// Maps spreadsheet columns by their header text so alternative templates
// (e.g. ones carrying a 知识内容 content column instead of / alongside
// 知识作者) are recognised. Falls back to the default positional layout
// when the headers cannot be matched.
const resolveColumns = (row) => {
  const defaults = {
    category_code: 0,
    category_name: 1,
    external_id: 2,
    article_code: 3,
    title: 4,
    author: 5,
    content: -1
  }
  if (!row || !isHeaderRow(row)) return defaults
  const match = (text, needles) => needles.some((n) => text.includes(n))
  const cols = { ...defaults }
  row.forEach((cell, idx) => {
    const text = cellText(cell).toLowerCase()
    if (!text) return
    if (match(text, ['分类code', '分类 code', 'categorycode', 'category code'])) cols.category_code = idx
    else if (match(text, ['分类名', '分类名称', 'categoryname', 'category name'])) cols.category_name = idx
    else if (match(text, ['知识id', '知识 id'])) cols.external_id = idx
    else if (match(text, ['知识code', '知识 code'])) cols.article_code = idx
    else if (match(text, ['标题'])) cols.title = idx
    else if (match(text, ['内容'])) cols.content = idx
    else if (match(text, ['作者'])) cols.author = idx
  })
  return cols
}

// Splits a combined "一级/二级" category value on the first "/" (or
// fullwidth "／") separator. Returns { parent, child }.
export const splitCategoryPath = (value) => {
  const v = cellText(value).replaceAll('／', '/')
  const idx = v.indexOf('/')
  if (idx < 0) return { parent: v, child: '' }
  return { parent: cellText(v.slice(0, idx)), child: cellText(v.slice(idx + 1)) }
}

// Parses the raw spreadsheet rows (array of arrays) into a flat list of
// knowledge import items ready to be sent to the backend. The second row
// is treated as a hint row (it is dropped silently) so the template can
// carry column explanations next to the header without polluting the
// import.
export const parseKnowledgeRows = (rows) => {
  const items = []
  let start = 0
  if (rows.length && isHeaderRow(rows[0])) start = 1
  const cols = resolveColumns(rows[0])
  const cell = (row, idx) => (idx >= 0 ? cellText(row?.[idx]) : '')
  // Hint row right after the header (matches what the template emits).
  if (rows.length > 1 && start === 1) {
    const hint = rows[1] || []
    const hintText = hint.map((c) => cellText(c)).join(' ')
    if (
      hintText.includes('必填') ||
      hintText.includes('一级') ||
      hintText.includes('二级') ||
      hintText.includes('required')
    ) {
      start = 2
    }
  }
  for (let i = start; i < rows.length; i++) {
    const row = rows[i] || []
    const rawName = cell(row, cols.category_name)
    const rawCode = cell(row, cols.category_code)
    // The 分类名 / 分类Code columns may carry a combined "一级/二级" path
    // (e.g. "其他（通用）知识库/游戏通用话术"). Split it so the tree renders
    // a proper two-level hierarchy. Values without a separator keep the
    // legacy template semantics: 分类Code = level-1 code, 分类名 = level-2
    // name.
    let item
    if (rawName.includes('/') || rawCode.includes('/')) {
      const nameParts = splitCategoryPath(rawName)
      const codeParts = splitCategoryPath(rawCode)
      item = {
        category_code: codeParts.child,
        category_name: nameParts.child,
        parent_category_code: codeParts.parent,
        parent_category_name: nameParts.parent
      }
    } else {
      item = {
        category_code: rawCode,
        category_name: rawName,
        parent_category_code: '',
        parent_category_name: ''
      }
    }
    item.external_id = cell(row, cols.external_id)
    item.article_code = cell(row, cols.article_code)
    item.title = cell(row, cols.title)
    item.author = cell(row, cols.author)
    item.content = cell(row, cols.content)
    // Drop totally blank rows but keep the row number so the user gets
    // line numbers in the error report.
    if (
      !item.category_code &&
      !item.category_name &&
      !item.parent_category_code &&
      !item.parent_category_name &&
      !item.external_id &&
      !item.article_code &&
      !item.title &&
      !item.author &&
      !item.content
    ) {
      continue
    }
    items.push({ ...item, _line: i + 1 })
  }
  return items
}

export const parseKnowledgeFile = async (file) => {
  const buffer = await file.arrayBuffer()
  const workbook = XLSX.read(buffer, { type: 'array' })
  const sheet = workbook.Sheets[workbook.SheetNames[0]]
  const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' })
  return parseKnowledgeRows(rows)
}

const writeSheet = (aoa, sheetName, fileName, colCount = 6) => {
  const widths = [18, 22, 18, 18, 36, 18, 60]
  const worksheet = XLSX.utils.aoa_to_sheet(aoa)
  worksheet['!cols'] = widths.slice(0, colCount).map((wch) => ({ wch }))
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, sheetName)
  XLSX.writeFile(workbook, fileName)
}

// Builds the template AOA: header row + hint row explaining each column,
// so the user can edit the file in place without leaving the example.
export const buildKnowledgeTemplate = (t) => {
  const aoa = [headerRow(t), HEADERS.map((h) => tr(t, `${h.key}Hint`, h.hint))]
  return aoa
}

// Downloads the empty import template.
export const downloadKnowledgeTemplate = (t) => {
  const aoa = buildKnowledgeTemplate(t)
  writeSheet(aoa, t('admin.knowledgeBase.templateSheetName'), 'knowledge-base-template.xlsx', HEADERS.length)
}

// Exports the full list of knowledge articles to a spreadsheet. Each row
// carries the combined "一级/二级" category path in the 分类名 column (the
// same format the importer accepts), and the article body is appended as a
// trailing column so exports round-trip losslessly.
export const exportKnowledgeArticles = (articles, t) => {
  const headers = headerRow(t)
  const aoa = [headers]
  for (const a of articles || []) {
    aoa.push([
      '',
      [a.parent_category_name, a.category_name].filter(Boolean).join('/'),
      a.external_id || '',
      a.code || '',
      a.title || '',
      a.author || '',
      a.content || ''
    ])
  }
  writeSheet(aoa, t('admin.knowledgeBase.templateSheetName'), 'knowledge-base.xlsx', headers.length)
}
