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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test } from 'vitest'

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { NoticePopupDialog } = await import('../notice-popup-dialog')
const { useNotificationStore } = await import('@/stores/notification-store')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'System Notice': 'System Notice',
        Confirm: 'Confirm',
      },
    },
  },
})

function renderPopup(options: { popupEnabled: boolean; notice: string }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  queryClient.setQueryData(['status'], {
    notice_popup_enabled: options.popupEnabled,
  })
  queryClient.setQueryData(['notice'], {
    success: true,
    data: options.notice,
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <NoticePopupDialog />
      </I18nextProvider>
    </QueryClientProvider>
  )
}

describe('notice popup dialog', () => {
  beforeEach(() => {
    useNotificationStore.setState({ popupConfirmedNotice: '' })
  })

  test('stays hidden when the administrator disabled the popup', () => {
    renderPopup({
      popupEnabled: false,
      notice: 'Scheduled maintenance tonight',
    })

    expect(screen.queryByText('System Notice')).toBeNull()
  })

  test('stays hidden when the notice is empty', () => {
    renderPopup({ popupEnabled: true, notice: '   ' })

    expect(screen.queryByText('System Notice')).toBeNull()
  })

  test('shows the notice and stops showing it after the user confirms', async () => {
    renderPopup({
      popupEnabled: true,
      notice: 'Scheduled maintenance tonight',
    })

    expect(screen.getByText('System Notice')).toBeInTheDocument()
    expect(
      screen.getByText('Scheduled maintenance tonight')
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(screen.queryByText('System Notice')).toBeNull()
    expect(useNotificationStore.getState().popupConfirmedNotice).toBe(
      'Scheduled maintenance tonight'
    )
  })

  test('shows again once the administrator edits the notice', () => {
    useNotificationStore.setState({
      popupConfirmedNotice: 'Scheduled maintenance tonight',
    })

    const unchanged = renderPopup({
      popupEnabled: true,
      notice: 'Scheduled maintenance tonight',
    })
    expect(screen.queryByText('System Notice')).toBeNull()
    unchanged.unmount()

    renderPopup({
      popupEnabled: true,
      notice: 'Maintenance postponed to Friday',
    })
    expect(screen.getByText('System Notice')).toBeInTheDocument()
    expect(
      screen.getByText('Maintenance postponed to Friday')
    ).toBeInTheDocument()
  })
})
