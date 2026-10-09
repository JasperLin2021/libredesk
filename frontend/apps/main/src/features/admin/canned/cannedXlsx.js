import * as XLSX from 'xlsx'

// Shared helpers for the Excel template / import / export of canned responses.
//
// The template uses three columns:
//   A: level-1 category (max 20 chars)
//   B: a canned response, OR a level-2 category name wrapped in [] (max 20 chars)
//   C: canned response (max 1000 chars)
//
// A row with a value in column A starts a new level-1 category. A plain value in
// column B is a canned response of the current level-1 category. A value in
// column B wrapped in [] is a level-2 category, whose responses live in column C.

const cellText = (value) => String(value ?? '').trim()

const isSubCategoryCell = (value) => {
  const s = cellText(value)
  return (s.startsWith('[') && s.endsWith(']')) || (s.startsWith('［') && s.endsWith('］'))
}

const subCategoryName = (value) => {
  const s = cellText(value)
  return s.slice(1, -1).trim()
}

// Detects whether a row is a header row, so that it can be skipped on import.
const looksLikeHeader = (row) => {
  if (!row) return false
  const joined = row.map((c) => cellText(c).toLowerCase()).join(' ')
  return joined.includes('分类') || joined.includes('category')
}

// Parses the raw spreadsheet rows (array of arrays) into a category tree plus
// the row numbers that were skipped because they had no owning category.
export const parseCannedRows = (rows) => {
  const errorRows = []
  const tree = []
  let currentRoot = null
  let currentSub = null

  let start = 0
  if (rows.length && looksLikeHeader(rows[0])) start = 1

  for (let i = start; i < rows.length; i++) {
    const row = rows[i] || []
    const rowNum = i + 1
    const a = cellText(row[0])
    const b = cellText(row[1])
    const c = cellText(row[2])
    if (!a && !b && !c) continue

    if (a) {
      currentRoot = { name: a, responses: [], children: [] }
      tree.push(currentRoot)
      currentSub = null
      if (b) {
        if (isSubCategoryCell(b)) {
          currentSub = { name: subCategoryName(b), responses: [], children: [] }
          currentRoot.children.push(currentSub)
          if (c) currentSub.responses.push(c)
        } else {
          currentRoot.responses.push(b)
        }
      }
      continue
    }

    if (b && isSubCategoryCell(b)) {
      if (!currentRoot) {
        errorRows.push(rowNum)
        continue
      }
      currentSub = { name: subCategoryName(b), responses: [], children: [] }
      currentRoot.children.push(currentSub)
      if (c) currentSub.responses.push(c)
      continue
    }

    if (b) {
      if (!currentRoot) {
        errorRows.push(rowNum)
        continue
      }
      currentRoot.responses.push(b)
      continue
    }

    if (c) {
      const target = currentSub || currentRoot
      if (!target) {
        errorRows.push(rowNum)
        continue
      }
      target.responses.push(c)
    }
  }

  return { tree, errorRows }
}

// Reads an uploaded File and returns the parsed category tree.
export const parseCannedFile = async (file) => {
  const buffer = await file.arrayBuffer()
  const workbook = XLSX.read(buffer, { type: 'array' })
  const sheet = workbook.Sheets[workbook.SheetNames[0]]
  const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' })
  return parseCannedRows(rows)
}

// Counts categories and responses in a parsed tree.
export const countCannedTree = (tree) => {
  let categories = 0
  let responses = 0
  const walk = (nodes) => {
    for (const node of nodes) {
      categories++
      responses += node.responses?.length || 0
      if (node.children?.length) walk(node.children)
    }
  }
  walk(tree)
  return { categories, responses }
}

// Builds the array-of-arrays representation used for both export and template
// generation. Only two levels are expressible in this format, so categories
// deeper than level 2 are flattened into their own level-2 rows.
const buildAoa = (tree, t) => {
  const aoa = [
    [
      t('admin.cannedResponses.templateCategoryColumn'),
      t('admin.cannedResponses.templateResponseColumn'),
      t('admin.cannedResponses.templateResponseColumn2')
    ]
  ]

  const emitResponses = (responses, column) => {
    for (const response of responses || []) {
      const row = ['', '', '']
      row[column] = response.content ?? response
      aoa.push(row)
    }
  }

  for (const root of tree) {
    aoa.push([root.name, '', ''])
    emitResponses(root.responses, 1)
    // Collect every descendant category (any depth) as a level-2 row.
    const descendants = []
    const collect = (children) => {
      for (const child of children || []) {
        descendants.push(child)
        if (child.children?.length) collect(child.children)
      }
    }
    collect(root.children)
    for (const child of descendants) {
      const responses = child.responses || []
      if (responses.length) {
        aoa.push(['', `[${child.name}]`, responses[0].content ?? responses[0]])
        emitResponses(responses.slice(1), 2)
      } else {
        aoa.push(['', `[${child.name}]`, ''])
      }
    }
  }
  return aoa
}

const writeSheet = (aoa, sheetName, fileName) => {
  const worksheet = XLSX.utils.aoa_to_sheet(aoa)
  worksheet['!cols'] = [{ wch: 22 }, { wch: 44 }, { wch: 34 }]
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, sheetName)
  XLSX.writeFile(workbook, fileName)
}

// Downloads the empty import template (with a few illustrative sample rows).
export const downloadCannedTemplate = (t) => {
  const sampleRoot = t('admin.cannedResponses.templateSampleRootCategory')
  const sampleResponse = (n) => t('admin.cannedResponses.templateSampleResponse', { n })
  const sampleSub = (n) => t('admin.cannedResponses.templateSampleSubCategory', { n })
  const aoa = [
    [
      t('admin.cannedResponses.templateCategoryColumn'),
      t('admin.cannedResponses.templateResponseColumn'),
      t('admin.cannedResponses.templateResponseColumn2')
    ],
    [sampleRoot, '', ''],
    ['', sampleResponse(1), ''],
    ['', sampleResponse(2), ''],
    ['', sampleResponse(3), ''],
    ['', `[${sampleSub(1)}]`, sampleResponse(1)],
    ['', `[${sampleSub(2)}]`, sampleResponse(2)]
  ]
  writeSheet(aoa, t('admin.cannedResponses.templateSheetName'), 'canned-responses-template.xlsx')
}

// Exports the current tree to an Excel file.
export const exportCannedTree = (tree, t) => {
  const aoa = buildAoa(tree, t)
  writeSheet(aoa, t('admin.cannedResponses.templateSheetName'), 'canned-responses.xlsx')
}
