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

import { getTransferFiles, selectTicketImages } from '../lib/attachments'
import { MAX_IMAGE_BYTES } from '../lib/limits'

function imageFile(name: string, type: string, size = 1024) {
  const file = new File(['x'], name, { type })
  Object.defineProperty(file, 'size', { value: size })
  return file
}

describe('selectTicketImages', () => {
  test('accepts PNG, JPEG, GIF and WebP images in their original order', () => {
    const incoming = [
      imageFile('a.png', 'image/png'),
      imageFile('b.jpg', 'image/jpeg'),
      imageFile('c.gif', 'image/gif'),
      imageFile('d.webp', 'image/webp'),
    ]

    const selection = selectTicketImages([], incoming)

    expect(selection.accepted).toEqual(incoming)
    expect(selection.rejected).toEqual([])
  })

  test.each([
    ['vector.svg', 'image/svg+xml'],
    ['photo.heic', 'image/heic'],
    ['page.html', 'text/html'],
    ['unknown', ''],
  ])('rejects %s as an unsupported type', (name, type) => {
    const file = imageFile(name, type)

    const selection = selectTicketImages([], [file])

    expect(selection.accepted).toEqual([])
    expect(selection.rejected).toEqual([{ file, reason: 'type' }])
  })

  test('accepts an image of exactly the size limit and rejects one byte more', () => {
    const atLimit = imageFile('max.png', 'image/png', MAX_IMAGE_BYTES)
    const overLimit = imageFile('big.png', 'image/png', MAX_IMAGE_BYTES + 1)

    const selection = selectTicketImages([], [atLimit, overLimit])

    expect(selection.accepted).toEqual([atLimit])
    expect(selection.rejected).toEqual([{ file: overLimit, reason: 'size' }])
  })

  test('fills only the remaining slots and rejects the rest by count', () => {
    const current = [
      imageFile('1.png', 'image/png'),
      imageFile('2.png', 'image/png'),
      imageFile('3.png', 'image/png'),
    ]
    const svg = imageFile('x.svg', 'image/svg+xml')
    const fourth = imageFile('4.png', 'image/png')
    const fifth = imageFile('5.png', 'image/png')

    const selection = selectTicketImages(current, [svg, fourth, fifth])

    expect(selection.accepted).toEqual([fourth])
    expect(selection.rejected).toEqual([
      { file: svg, reason: 'type' },
      { file: fifth, reason: 'count' },
    ])
  })

  test('rejects every image once the message already has four', () => {
    const current = ['1', '2', '3', '4'].map((name) =>
      imageFile(`${name}.png`, 'image/png')
    )
    const extra = imageFile('5.png', 'image/png')

    const selection = selectTicketImages(current, [extra])

    expect(selection.accepted).toEqual([])
    expect(selection.rejected).toEqual([{ file: extra, reason: 'count' }])
  })
})

describe('getTransferFiles', () => {
  test('takes file items and ignores text items of a paste', () => {
    const screenshot = imageFile('image.png', 'image/png')
    const clipboard = {
      items: [
        { kind: 'string', getAsFile: () => null },
        { kind: 'file', getAsFile: () => screenshot },
      ],
      files: [],
    } as unknown as DataTransfer

    expect(getTransferFiles(clipboard)).toEqual([screenshot])
  })

  test('falls back to the file list when no file items are exposed', () => {
    const dropped = imageFile('drop.jpg', 'image/jpeg')
    const transfer = { items: [], files: [dropped] } as unknown as DataTransfer

    expect(getTransferFiles(transfer)).toEqual([dropped])
  })

  test('returns nothing for a missing transfer', () => {
    expect(getTransferFiles(null)).toEqual([])
  })
})
