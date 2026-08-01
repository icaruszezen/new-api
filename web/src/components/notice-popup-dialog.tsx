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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'
import { getNotice } from '@/lib/api'
import { useNotificationStore } from '@/stores/notification-store'

/**
 * Shows the system notice in a centered dialog when the administrator enabled
 * the auto popup. Confirming stores the notice content, so the dialog only
 * returns once the administrator edits the notice.
 */
export function NoticePopupDialog() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const popupEnabled = status?.notice_popup_enabled === true

  const { data: noticeResponse } = useQuery({
    queryKey: ['notice'],
    queryFn: getNotice,
    staleTime: 1000 * 60 * 5,
    enabled: popupEnabled,
  })

  const popupConfirmedNotice = useNotificationStore(
    (state) => state.popupConfirmedNotice
  )
  const confirmNoticePopup = useNotificationStore(
    (state) => state.confirmNoticePopup
  )

  const notice = noticeResponse?.success
    ? (noticeResponse.data || '').trim()
    : ''
  const open = popupEnabled && notice !== '' && notice !== popupConfirmedNotice

  return (
    <Dialog
      open={open}
      // Escape and outside presses must not count as an acknowledgement,
      // so only the confirm handler is allowed to close the dialog.
      onOpenChange={() => undefined}
      disablePointerDismissal
      showCloseButton={false}
      title={t('System Notice')}
      contentClassName='sm:max-w-lg'
      footer={
        <Button onClick={() => confirmNoticePopup(notice)}>
          {t('Confirm')}
        </Button>
      }
    >
      <RichContent breaks content={notice} />
    </Dialog>
  )
}
