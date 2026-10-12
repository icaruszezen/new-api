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
import { sideDrawerContentClassName } from '@/components/drawer-layout'
import { Sheet, SheetContent } from '@/components/ui/sheet'

import { TicketDetailPanel } from './ticket-detail-panel'

type TicketDetailSheetProps = {
  /** Kept after closing so the panel stays filled during the exit animation. */
  ticketId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TicketDetailSheet(props: TicketDetailSheetProps) {
  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-2xl')}>
        {props.ticketId !== null && (
          <TicketDetailPanel key={props.ticketId} ticketId={props.ticketId} />
        )}
      </SheetContent>
    </Sheet>
  )
}
