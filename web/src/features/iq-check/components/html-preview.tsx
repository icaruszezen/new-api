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
import { Code, Eye, Maximize2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

/** Start loading a frame shortly before it scrolls into view. */
const PREVIEW_ROOT_MARGIN = '300px'

/**
 * Model-generated pages are untrusted. Scripts may run so interactive answers
 * can be judged, but the frame gets an opaque origin (no allow-same-origin),
 * so it cannot read this site's cookies or storage, navigate the top window,
 * open popups, submit forms or block the page with modal dialogs.
 */
function PreviewFrame(props: { html: string; title: string }) {
  return (
    <iframe
      title={props.title}
      srcDoc={props.html}
      sandbox='allow-scripts'
      referrerPolicy='no-referrer'
      loading='lazy'
      className='block size-full border-0 bg-white'
    />
  )
}

type HtmlPreviewProps = {
  html: string
  /** False when the answer stopped before the page was closed. */
  complete: boolean
  /** Taller frame without the enlarge action, used inside the output dialog. */
  expanded?: boolean
}

export function HtmlPreview(props: HtmlPreviewProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement>(null)
  const [showSource, setShowSource] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  // Pages can be heavy, so a frame is only created once its box is close to
  // the viewport. Environments without IntersectionObserver render at once.
  const [nearViewport, setNearViewport] = useState(
    () => typeof IntersectionObserver === 'undefined'
  )

  useEffect(() => {
    if (nearViewport) return
    const element = containerRef.current
    if (!element) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setNearViewport(true)
          observer.disconnect()
        }
      },
      { rootMargin: PREVIEW_ROOT_MARGIN }
    )
    observer.observe(element)
    return () => observer.disconnect()
  }, [nearViewport])

  const previewTitle = t('HTML preview')

  return (
    <div
      ref={containerRef}
      className='border-border/70 overflow-hidden rounded-lg border'
    >
      <div className='bg-muted/40 border-border/70 flex flex-wrap items-center gap-2 border-b px-2.5 py-1.5'>
        <span className='text-muted-foreground text-xs font-medium'>
          {previewTitle}
        </span>
        {!props.complete && (
          <Badge
            variant='warning'
            title={t('The output ended before the page was finished.')}
          >
            {t('Incomplete')}
          </Badge>
        )}
        <div className='ml-auto flex items-center gap-1'>
          <Button
            type='button'
            variant='ghost'
            size='xs'
            aria-pressed={showSource}
            onClick={() => setShowSource((current) => !current)}
          >
            {showSource ? (
              <Eye aria-hidden='true' />
            ) : (
              <Code aria-hidden='true' />
            )}
            {showSource ? t('View preview') : t('View source')}
          </Button>
          {!props.expanded && (
            <Button
              type='button'
              variant='ghost'
              size='xs'
              onClick={() => setDialogOpen(true)}
            >
              <Maximize2 aria-hidden='true' />
              {t('Open larger')}
            </Button>
          )}
        </div>
      </div>

      <div className={props.expanded ? 'h-[60vh]' : 'h-72'}>
        {showSource && (
          <pre className='size-full overflow-auto p-3 font-mono text-xs break-all whitespace-pre-wrap'>
            {props.html}
          </pre>
        )}
        {!showSource && nearViewport && (
          <PreviewFrame html={props.html} title={previewTitle} />
        )}
      </div>

      {!props.expanded && (
        <Dialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          title={previewTitle}
          contentClassName='w-[90vw] sm:max-w-[90vw]'
        >
          <div className='border-border/70 h-[min(80vh,calc(100vh-15rem))] overflow-hidden rounded-lg border'>
            {dialogOpen && (
              <PreviewFrame html={props.html} title={previewTitle} />
            )}
          </div>
        </Dialog>
      )}
    </div>
  )
}
