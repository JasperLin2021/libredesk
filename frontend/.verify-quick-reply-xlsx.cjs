/* eslint-disable */
// Temporary verification script: generates sample Excel files (matching the
// frontend export format) and validates that the frontend parse logic (copied
// verbatim from TopicQuestionManager.vue) correctly parses them.
const XLSX = require('xlsx')
const assert = require('assert')

// ---- Frontend parse logic (mirrors TopicQuestionManager.vue) ----
const normalizeHeader = (cell) => {
  const s = String(cell ?? '').trim().toLowerCase()
  if (['topic name', '主题名称', '主题'].includes(s)) return 'name'
  if (['aliases', '别名'].includes(s)) return 'aliases'
  if (['hint message', 'hint', '提示语', '提示文案'].includes(s)) return 'hint_message'
  if (['question', '问题'].includes(s)) return 'question'
  if (['answer', '答案'].includes(s)) return 'answer'
  return null
}

function parseLikeFrontend(filePath) {
  const workbook = XLSX.readFile(filePath)
  const sheet = workbook.Sheets[workbook.SheetNames[0]]
  const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' })
  if (!rows.length) return { topics: [], errorRows: [], invalid: true }

  const headerMap = new Map()
  rows[0].forEach((cell, index) => {
    const key = normalizeHeader(cell)
    if (key && !headerMap.has(key)) headerMap.set(key, index)
  })
  const nameIdx = headerMap.get('name')
  const aliasesIdx = headerMap.get('aliases')
  const hintIdx = headerMap.get('hint_message')
  const questionIdx = headerMap.get('question')
  const answerIdx = headerMap.get('answer')
  if (nameIdx === undefined || questionIdx === undefined) {
    return { topics: [], errorRows: [], invalid: true }
  }

  const topicMap = new Map()
  const order = []
  const errorRows = []
  for (let i = 1; i < rows.length; i++) {
    const rowNum = i + 1
    const row = rows[i]
    const name = nameIdx >= 0 ? String(row[nameIdx] ?? '').trim() : ''
    const question = questionIdx >= 0 ? String(row[questionIdx] ?? '').trim() : ''
    if (!name && !question) continue // skip fully empty rows
    if (!name) {
      errorRows.push(rowNum)
      continue
    }
    const rawAliases = aliasesIdx >= 0 ? String(row[aliasesIdx] ?? '') : ''
    const aliases = rawAliases.split(/[;；,，]/).map((a) => a.trim()).filter(Boolean)
    const hint = hintIdx >= 0 ? String(row[hintIdx] ?? '').trim() : ''
    const answer = answerIdx >= 0 ? String(row[answerIdx] ?? '') : ''

    let topic = topicMap.get(name)
    if (!topic) {
      topic = { name, aliases: new Set(), hint_message: hint, questions: [] }
      topicMap.set(name, topic)
      order.push(topic)
    }
    aliases.forEach((a) => topic.aliases.add(a))
    if (!topic.hint_message && hint) topic.hint_message = hint
    if (question) topic.questions.push({ question, answer })
  }

  const topics = order.map((t) => ({
    name: t.name,
    aliases: [...t.aliases],
    hint_message: t.hint_message,
    questions: t.questions,
  }))
  return { topics, errorRows }
}

// ---- Sample data ----
const zhHeaders = ['主题名称', '别名', '提示语', '问题', '答案']
const zhRows = [
  ['发货', '物流;快递', '回复客户发货相关咨询', '你们发什么快递？', '默认发顺丰或圆通。'],
  ['发货', '物流;快递', '', '几天能到？', '一般 3-5 个工作日。'],
  ['退货', '退款,售后', '处理退货退款相关咨询', '怎么申请退货？', '在订单页面点击退货即可。'],
  ['无问题主题', '', '', '', ''],
  ['', '别名', '', '缺少主题名称：应被跳过', ''],
  ['', '', '', '', ''],
]

const enHeaders = ['Topic name', 'Aliases', 'Hint message', 'Question', 'Answer']
const enRows = [
  ['Shipping', 'logistics;courier', 'Replies about shipping', 'Which courier do you use?', 'SF Express or YTO.'],
  ['Shipping', 'logistics;courier', '', 'How long does delivery take?', 'Usually 3-5 business days.'],
  ['Returns', 'refund,after-sales', 'Replies about returns', 'How do I request a return?', 'Click Return on the order page.'],
  ['No-questions topic', '', '', '', ''],
  ['', 'alias', '', 'Missing topic name: should be skipped', ''],
  ['', '', '', '', ''],
]

function buildSheet(headers, rows) {
  const aoa = [headers, ...rows]
  const worksheet = XLSX.utils.aoa_to_sheet(aoa)
  worksheet['!cols'] = [{ wch: 24 }, { wch: 20 }, { wch: 30 }, { wch: 30 }, { wch: 40 }]
  return worksheet
}

function writeSample(filePath, headers, rows) {
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, buildSheet(headers, rows), 'Quick Replies')
  XLSX.writeFile(workbook, filePath)
}

// ---- Generate + verify ----
const zhFile = 'e:/libredesk/quick-reply-import-sample-zh.xlsx'
const enFile = 'e:/libredesk/quick-reply-import-sample-en.xlsx'
writeSample(zhFile, zhHeaders, zhRows)
writeSample(enFile, enHeaders, enRows)

for (const [label, file, expectedTopics, expectedErrors] of [
  ['zh', zhFile, [
    { name: '发货', aliases: ['物流', '快递'], questions: 2, hint: '回复客户发货相关咨询' },
    { name: '退货', aliases: ['退款', '售后'], questions: 1, hint: '处理退货退款相关咨询' },
    { name: '无问题主题', aliases: [], questions: 0, hint: '' },
  ], [6]],
  ['en', enFile, [
    { name: 'Shipping', aliases: ['logistics', 'courier'], questions: 2, hint: 'Replies about shipping' },
    { name: 'Returns', aliases: ['refund', 'after-sales'], questions: 1, hint: 'Replies about returns' },
    { name: 'No-questions topic', aliases: [], questions: 0, hint: '' },
  ], [6]],
]) {
  const { topics, errorRows, invalid } = parseLikeFrontend(file)
  assert.ok(!invalid, `${label}: header should be recognized`)
  assert.deepStrictEqual(errorRows, expectedErrors, `${label}: error rows mismatch`)
  assert.strictEqual(topics.length, expectedTopics.length, `${label}: topic count mismatch`)
  expectedTopics.forEach((expected, i) => {
    const actual = topics[i]
    assert.strictEqual(actual.name, expected.name, `${label}: topic name mismatch`)
    assert.deepStrictEqual(actual.aliases, expected.aliases, `${label}: aliases mismatch for ${expected.name}`)
    assert.strictEqual(actual.questions.length, expected.questions, `${label}: question count mismatch for ${expected.name}`)
    assert.strictEqual(actual.hint_message, expected.hint, `${label}: hint mismatch for ${expected.name}`)
  })
  console.log(`[OK] ${label}: ${topics.length} topics parsed, error rows: ${errorRows.join(',') || 'none'}`)
}

// Round-trip: exported rows (as produced by exportTopics) parse back identically.
const exportAoa = [
  ['主题名称', '别名', '提示语', '问题', '答案'],
  ['发货', '物流; 快递', '提示', '问题一', '答案一'],
  ['发货', '物流; 快递', '', '问题二', '答案二'],
  ['无问题主题', '', '', '', ''],
]
const rtFile = 'e:/libredesk/.verify-roundtrip.xlsx'
const rtWorkbook = XLSX.utils.book_new()
XLSX.utils.book_append_sheet(rtWorkbook, XLSX.utils.aoa_to_sheet(exportAoa), 'Quick Replies')
XLSX.writeFile(rtWorkbook, rtFile)
const rt = parseLikeFrontend(rtFile)
assert.strictEqual(rt.topics.length, 2, 'roundtrip: topic count')
assert.strictEqual(rt.topics[0].questions.length, 2, 'roundtrip: question count for 发货')
assert.strictEqual(rt.topics[1].questions.length, 0, 'roundtrip: topic without questions kept')
console.log('[OK] round-trip: exported file parses back correctly')

console.log('All verification checks passed.')
