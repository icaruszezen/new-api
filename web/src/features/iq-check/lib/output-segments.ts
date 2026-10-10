/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
/**
 * A model answer split into what is shown as plain text and what is rendered
 * as a web page. `complete` is false when the answer stopped before the HTML
 * was closed (for example when max tokens cut the output short).
 */
export type OutputSegment =
  | { type: 'text'; text: string }
  | { type: 'html'; html: string; complete: boolean }

type FenceOpening = {
  marker: '`' | '~'
  length: number
  info: string
}

const FENCE_OPEN_PATTERN = /^ {0,3}(`{3,}|~{3,})(.*)$/
const FENCE_CLOSE_PATTERN = /^ {0,3}(`{3,}|~{3,})[ \t]*$/
const HTML_FENCE_LANGUAGES = new Set(['html', 'htm', 'xhtml'])
const HTML_DOCUMENT_PREFIX_PATTERN = /^(?:<!doctype\s+html|<html(?=[\s>/]|$))/i
const RAW_DOCUMENT_START_PATTERN = /<!doctype\s+html|<html(?=[\s>/])/gi
const RAW_DOCUMENT_END_PATTERN = /<\/html\s*>/i
const WHITESPACE_ONLY_PATTERN = /^\s*$/

function parseFenceOpening(line: string): FenceOpening | null {
  const match = FENCE_OPEN_PATTERN.exec(line)
  if (!match) return null
  const marker = match[1][0] === '`' ? '`' : '~'
  const info = match[2].trim()
  // CommonMark: a backtick fence's info string cannot contain backticks, so
  // a line such as ```js `x` ``` is inline code rather than a block.
  if (marker === '`' && info.includes('`')) return null
  return { marker, length: match[1].length, info }
}

function closesFence(line: string, opening: FenceOpening): boolean {
  const match = FENCE_CLOSE_PATTERN.exec(line)
  return (
    match !== null &&
    match[1][0] === opening.marker &&
    match[1].length >= opening.length
  )
}

function isHtmlFence(info: string, body: string): boolean {
  const language = info.split(/\s+/)[0].toLowerCase()
  if (HTML_FENCE_LANGUAGES.has(language)) return true
  return info === '' && HTML_DOCUMENT_PREFIX_PATTERN.test(body.trimStart())
}

function appendText(segments: OutputSegment[], text: string) {
  if (text === '') return
  const last = segments.at(-1)
  if (last?.type === 'text') {
    last.text += text
    return
  }
  segments.push({ type: 'text', text })
}

/**
 * Splits prose into text and raw HTML documents. A document starts at a
 * doctype or an `<html>` tag and ends after the first `</html>`; without one it
 * runs to the end of the prose. Tags quoted in inline code (`<html>`) are
 * explanations, not documents, so they are left in the text.
 */
function appendProse(segments: OutputSegment[], prose: string) {
  let cursor = 0
  for (const start of prose.matchAll(RAW_DOCUMENT_START_PATTERN)) {
    if (start.index < cursor) continue
    if (start.index > 0 && prose[start.index - 1] === '`') continue

    appendText(segments, prose.slice(cursor, start.index))
    const document = prose.slice(start.index)
    const end = RAW_DOCUMENT_END_PATTERN.exec(document)
    if (!end) {
      segments.push({ type: 'html', html: document, complete: false })
      return
    }
    const documentEnd = end.index + end[0].length
    segments.push({
      type: 'html',
      html: document.slice(0, documentEnd),
      complete: true,
    })
    cursor = start.index + documentEnd
  }
  appendText(segments, prose.slice(cursor))
}

/**
 * Finds the HTML pages inside a model answer so they can be previewed while
 * everything else stays readable text.
 *
 * Fenced blocks are resolved first: ```html / ```htm / ```xhtml blocks, and
 * unlabeled blocks whose body is an HTML document, become previews holding
 * only the block body. Any other fenced block is kept verbatim as text and is
 * never searched for HTML, so code samples that mention `<html>` stay code.
 * The remaining prose is then searched for raw HTML documents.
 */
export function parseOutputSegments(content: string): OutputSegment[] {
  const normalized = content.replaceAll(/\r\n?/g, '\n')
  const lines = normalized.split('\n')
  // lineStarts[i] is the offset of line i; the extra entry marks the end.
  const lineStarts: number[] = []
  let offset = 0
  for (const line of lines) {
    lineStarts.push(offset)
    offset += line.length + 1
  }
  lineStarts.push(offset)

  const segments: OutputSegment[] = []
  let proseStart = 0
  let index = 0
  while (index < lines.length) {
    const opening = parseFenceOpening(lines[index])
    if (!opening) {
      index += 1
      continue
    }

    let closeIndex = index + 1
    while (
      closeIndex < lines.length &&
      !closesFence(lines[closeIndex], opening)
    ) {
      closeIndex += 1
    }
    const complete = closeIndex < lines.length
    const blockStart = lineStarts[index]
    const blockEnd = complete ? lineStarts[closeIndex + 1] : normalized.length

    appendProse(segments, normalized.slice(proseStart, blockStart))
    const body = lines.slice(index + 1, closeIndex).join('\n')
    if (!isHtmlFence(opening.info, body)) {
      appendText(segments, normalized.slice(blockStart, blockEnd))
    } else if (!WHITESPACE_ONLY_PATTERN.test(body)) {
      segments.push({ type: 'html', html: body, complete })
    }

    proseStart = blockEnd
    index = complete ? closeIndex + 1 : lines.length
  }
  appendProse(segments, normalized.slice(proseStart))

  return segments.flatMap((segment): OutputSegment[] => {
    if (segment.type === 'html') return [segment]
    if (WHITESPACE_ONLY_PATTERN.test(segment.text)) return []
    // Blank lines around a block only separated it from its neighbours.
    const text = segment.text
      .replace(/^(?:[ \t]*\n)+/, '')
      .replace(/(?:\n[ \t]*)+$/, '')
    return [{ type: 'text', text }]
  })
}
