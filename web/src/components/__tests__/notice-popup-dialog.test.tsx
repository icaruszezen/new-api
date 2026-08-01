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
import assert from 'node:assert/strict'
import { after, beforeEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'localStorage',
  'sessionStorage',
  'matchMedia',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'DocumentFragment',
  'Event',
  'CustomEvent',
  'KeyboardEvent',
  'MouseEvent',
  'PointerEvent',
  'MutationObserver',
  'ResizeObserver',
  'IntersectionObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')

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

const { NoticePopupDialog } = await import('../notice-popup-dialog')
const { useNotificationStore } = await import('@/stores/notification-store')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type Rendered = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
  queryClient: InstanceType<typeof QueryClient>
}

async function renderPopup(options: {
  popupEnabled: boolean
  notice: string
}): Promise<Rendered> {
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

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <NoticePopupDialog />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, root, queryClient }
}

async function unmountPopup(rendered: Rendered) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
  rendered.queryClient.clear()
}

function findConfirmButton(): HTMLElement | null {
  const buttons = [...document.body.querySelectorAll('button')]
  return (
    buttons.find((button) => button.textContent?.trim() === 'Confirm') ?? null
  )
}

function popupIsVisible(): boolean {
  return document.body.textContent?.includes('System Notice') === true
}

describe('notice popup dialog', () => {
  beforeEach(() => {
    useNotificationStore.setState({ popupConfirmedNotice: '' })
    document.body.innerHTML = ''
  })

  after(() => {
    domWindow.close()
  })

  test('stays hidden when the administrator disabled the popup', async () => {
    const rendered = await renderPopup({
      popupEnabled: false,
      notice: 'Scheduled maintenance tonight',
    })

    assert.equal(popupIsVisible(), false)

    await unmountPopup(rendered)
  })

  test('stays hidden when the notice is empty', async () => {
    const rendered = await renderPopup({ popupEnabled: true, notice: '   ' })

    assert.equal(popupIsVisible(), false)

    await unmountPopup(rendered)
  })

  test('shows the notice and stops showing it after the user confirms', async () => {
    const rendered = await renderPopup({
      popupEnabled: true,
      notice: 'Scheduled maintenance tonight',
    })

    assert.equal(popupIsVisible(), true)
    assert.equal(
      document.body.textContent?.includes('Scheduled maintenance tonight'),
      true
    )

    const confirmButton = findConfirmButton()
    assert.ok(confirmButton)
    await act(async () => {
      confirmButton.click()
    })

    assert.equal(popupIsVisible(), false)
    assert.equal(
      useNotificationStore.getState().popupConfirmedNotice,
      'Scheduled maintenance tonight'
    )

    await unmountPopup(rendered)
  })

  test('shows again once the administrator edits the notice', async () => {
    useNotificationStore.setState({
      popupConfirmedNotice: 'Scheduled maintenance tonight',
    })

    const unchanged = await renderPopup({
      popupEnabled: true,
      notice: 'Scheduled maintenance tonight',
    })
    assert.equal(popupIsVisible(), false)
    await unmountPopup(unchanged)

    const edited = await renderPopup({
      popupEnabled: true,
      notice: 'Maintenance postponed to Friday',
    })
    assert.equal(popupIsVisible(), true)
    assert.equal(
      document.body.textContent?.includes('Maintenance postponed to Friday'),
      true
    )
    await unmountPopup(edited)
  })
})
