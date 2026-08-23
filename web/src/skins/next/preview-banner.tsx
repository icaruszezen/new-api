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
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { useUserConsolePreview } from '../use-user-console-preview'

/**
 * Fixed overlay so the classic console height math (`100svh - header`) is
 * left untouched. Regular users on the next shell never see this.
 */
export function NextConsolePreviewBanner() {
  const { t } = useTranslation()
  const preview = useUserConsolePreview()

  if (!preview.isPreviewing) return null

  return (
    <div
      role='status'
      className='bg-primary text-primary-foreground fixed inset-x-0 top-0 z-50 flex h-9 items-center justify-center gap-3 px-4 text-sm'
    >
      <span>{t('You are previewing the user console.')}</span>
      <Button
        type='button'
        variant='secondary'
        size='xs'
        onClick={preview.stopPreview}
      >
        {t('Exit preview')}
      </Button>
    </div>
  )
}
