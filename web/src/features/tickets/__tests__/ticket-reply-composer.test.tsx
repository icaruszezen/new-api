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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { TicketReplyComposer } from '../components/ticket-reply-composer'

// jsdom has no object URL support; the picker previews need it.
const objectUrlApi = {
  createObjectURL: Object.getOwnPropertyDescriptor(URL, 'createObjectURL'),
  revokeObjectURL: Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL'),
}

beforeEach(() => {
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(() => 'blob:preview'),
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(),
  })
})

afterEach(() => {
  for (const [name, descriptor] of Object.entries(objectUrlApi)) {
    if (descriptor) {
      Object.defineProperty(URL, name, descriptor)
    } else {
      Reflect.deleteProperty(URL, name)
    }
  }
})

function renderComposer(onSubmit: () => Promise<boolean>) {
  const view = render(<TicketReplyComposer onSubmit={onSubmit} />)
  const fileInput =
    view.container.querySelector<HTMLInputElement>('input[type="file"]')
  if (!fileInput) throw new Error('file input not rendered')
  return { fileInput }
}

describe('TicketReplyComposer', () => {
  test('keeps submit disabled until the reply has text or an image', async () => {
    const user = userEvent.setup()
    const { fileInput } = renderComposer(vi.fn())
    const submit = screen.getByRole('button', { name: 'Send reply' })

    expect(submit).toBeDisabled()

    await user.type(screen.getByRole('textbox', { name: 'Reply' }), '   ')
    expect(submit).toBeDisabled()

    await user.upload(
      fileInput,
      new File(['png'], 'shot.png', { type: 'image/png' })
    )
    expect(submit).toBeEnabled()
  })

  test('submits the trimmed text with the images and clears the draft after success', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn().mockResolvedValue(true)
    const { fileInput } = renderComposer(onSubmit)
    const image = new File(['png'], 'shot.png', { type: 'image/png' })
    const textbox = screen.getByRole('textbox', { name: 'Reply' })

    await user.type(textbox, '  It still fails  ')
    await user.upload(fileInput, image)
    await user.click(screen.getByRole('button', { name: 'Send reply' }))

    expect(onSubmit).toHaveBeenCalledWith({
      content: 'It still fails',
      files: [image],
    })
    await waitFor(() => expect(textbox).toHaveValue(''))
    expect(screen.queryByRole('button', { name: 'Remove image' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Send reply' })).toBeDisabled()
  })

  test('keeps the draft when the submission is not accepted', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn().mockResolvedValue(false)
    const { fileInput } = renderComposer(onSubmit)
    const textbox = screen.getByRole('textbox', { name: 'Reply' })

    await user.type(textbox, 'Please check')
    await user.upload(
      fileInput,
      new File(['png'], 'shot.png', { type: 'image/png' })
    )
    await user.click(screen.getByRole('button', { name: 'Send reply' }))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Send reply' })).toBeEnabled()
    )
    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(textbox).toHaveValue('Please check')
    expect(
      screen.getByRole('button', { name: 'Remove image' })
    ).toBeInTheDocument()
  })
})
