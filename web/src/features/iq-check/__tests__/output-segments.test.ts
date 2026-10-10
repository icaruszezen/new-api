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
import { describe, expect, test } from 'vitest'

import { parseOutputSegments } from '../lib/output-segments'

const PAGE = '<!DOCTYPE html>\n<html>\n<body><h1>Hi</h1></body>\n</html>'

describe('parseOutputSegments', () => {
  test.each(['', '   ', '\n\n  \n'])(
    'returns no segments for empty output %j',
    (content) => {
      expect(parseOutputSegments(content)).toEqual([])
    }
  )

  test('keeps output without HTML as one text segment', () => {
    expect(parseOutputSegments('\n\nThe answer is 42.\nDone.\n\n')).toEqual([
      { type: 'text', text: 'The answer is 42.\nDone.' },
    ])
  })

  test('renders an html fence between prose as a preview of its body only', () => {
    const content = [
      'Here is the page:',
      '',
      '```html',
      '<div>Clock</div>',
      '```',
      '',
      'Open it in a browser.',
    ].join('\n')

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Here is the page:' },
      { type: 'html', html: '<div>Clock</div>', complete: true },
      { type: 'text', text: 'Open it in a browser.' },
    ])
  })

  test.each(['HTML', 'Htm', 'xhtml', 'html title="page"'])(
    'treats a ```%s fence as HTML regardless of case or extra info',
    (info) => {
      const content = `\`\`\`${info}\n<p>x</p>\n\`\`\``

      expect(parseOutputSegments(content)).toEqual([
        { type: 'html', html: '<p>x</p>', complete: true },
      ])
    }
  )

  test('treats an unlabeled fence holding an HTML document as HTML', () => {
    const content = `\`\`\`\n  ${PAGE}\n\`\`\``

    expect(parseOutputSegments(content)).toEqual([
      { type: 'html', html: `  ${PAGE}`, complete: true },
    ])
  })

  test('keeps an unlabeled fence of plain markup as text', () => {
    const content = '```\n<div>not a page</div>\n```'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: content },
    ])
  })

  test('keeps a non-HTML fence verbatim and does not search it for documents', () => {
    const content = [
      'Use this template:',
      '```js',
      'const page = `<html><body></body></html>`',
      '```',
    ].join('\n')

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: content },
    ])
  })

  test('extracts a raw document from the middle of the text', () => {
    const content = `Sure! ${PAGE}\nHope this helps.`

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Sure! ' },
      { type: 'html', html: PAGE, complete: true },
      { type: 'text', text: 'Hope this helps.' },
    ])
  })

  test('extracts a raw document that starts with <html> and no doctype', () => {
    const content =
      'Result:\n<HTML lang="en"><body>ok</body></HTML>\nThat is all.'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Result:' },
      {
        type: 'html',
        html: '<HTML lang="en"><body>ok</body></HTML>',
        complete: true,
      },
      { type: 'text', text: 'That is all.' },
    ])
  })

  test('does not mistake tags such as <htmlfoo> or inline `<html>` for documents', () => {
    const content = 'The `<html>` element wraps <htmlfoo> nodes.'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: content },
    ])
  })

  test('marks an unclosed html fence as incomplete and keeps the rest as its body', () => {
    const content = 'Page:\n```html\n<html><body>\n<p>cut off'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Page:' },
      {
        type: 'html',
        html: '<html><body>\n<p>cut off',
        complete: false,
      },
    ])
  })

  test('marks a raw document without </html> as incomplete', () => {
    const content = 'Page:\n<!doctype html><html><body><p>cut off'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Page:' },
      {
        type: 'html',
        html: '<!doctype html><html><body><p>cut off',
        complete: false,
      },
    ])
  })

  test('ends an unclosed raw document at the next fenced block', () => {
    const content = '<html><body>draft\n```css\nbody {}\n```'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'html', html: '<html><body>draft\n', complete: false },
      { type: 'text', text: '```css\nbody {}\n```' },
    ])
  })

  test('returns several HTML blocks in their original order', () => {
    const content = [
      'First:',
      '```html',
      '<b>one</b>',
      '```',
      'Second:',
      '<html><i>two</i></html>',
      'Third:',
      '~~~html',
      '<u>three</u>',
      '~~~',
    ].join('\n')

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'First:' },
      { type: 'html', html: '<b>one</b>', complete: true },
      { type: 'text', text: 'Second:' },
      { type: 'html', html: '<html><i>two</i></html>', complete: true },
      { type: 'text', text: 'Third:' },
      { type: 'html', html: '<u>three</u>', complete: true },
    ])
  })

  test('only closes a fence with the same marker at least as long as the opener', () => {
    const content = ['````html', '<pre>', '```', '~~~~', '</pre>', '````'].join(
      '\n'
    )

    expect(parseOutputSegments(content)).toEqual([
      {
        type: 'html',
        html: '<pre>\n```\n~~~~\n</pre>',
        complete: true,
      },
    ])
  })

  test('merges prose around a non-HTML fence into one text segment', () => {
    const content = 'Before\n~~~python\nprint(1)\n~~~\nAfter'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: content },
    ])
  })

  test('normalizes Windows line endings before scanning', () => {
    const content = 'Intro\r\n```html\r\n<p>x</p>\r\n```\r\nOutro'

    expect(parseOutputSegments(content)).toEqual([
      { type: 'text', text: 'Intro' },
      { type: 'html', html: '<p>x</p>', complete: true },
      { type: 'text', text: 'Outro' },
    ])
  })

  test('drops an html fence whose body is empty', () => {
    expect(parseOutputSegments('Nothing:\n```html\n\n```\nEnd')).toEqual([
      { type: 'text', text: 'Nothing:\nEnd' },
    ])
  })
})
