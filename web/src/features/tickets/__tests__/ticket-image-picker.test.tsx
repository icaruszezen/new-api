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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { TicketImagePicker } from '../components/ticket-image-picker'

// jsdom has no object URL support; previews need it.
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

function png(name: string) {
  return new File(['png'], name, { type: 'image/png' })
}

describe('TicketImagePicker', () => {
  test('adds supported images and toasts instead of adding an SVG', async () => {
    const user = userEvent.setup({ applyAccept: false })
    const toastError = vi.spyOn(toast, 'error').mockReturnValue('toast')
    const onChange = vi.fn()
    const image = png('shot.png')
    const view = render(<TicketImagePicker files={[]} onChange={onChange} />)
    const input =
      view.container.querySelector<HTMLInputElement>('input[type="file"]')
    if (!input) throw new Error('file input not rendered')

    await user.upload(input, [
      new File(['<svg/>'], 'logo.svg', { type: 'image/svg+xml' }),
      image,
    ])

    expect(onChange).toHaveBeenCalledWith([image])
    expect(toastError).toHaveBeenCalledWith(
      'Only PNG, JPEG, GIF and WebP images are supported'
    )
  })

  test('disables adding once four images are attached and removes by button', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const files = [png('1.png'), png('2.png'), png('3.png'), png('4.png')]
    render(<TicketImagePicker files={files} onChange={onChange} />)

    expect(screen.getByRole('button', { name: 'Add images' })).toBeDisabled()
    expect(screen.getByText('4/4 images')).toBeInTheDocument()

    await user.click(screen.getAllByRole('button', { name: 'Remove image' })[1])

    expect(onChange).toHaveBeenCalledWith([files[0], files[2], files[3]])
  })
})
