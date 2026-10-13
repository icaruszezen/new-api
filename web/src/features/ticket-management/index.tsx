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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { getAdminTickets, getAdminTicketStats } from '@/features/tickets/api'
import { TICKET_QUERY_KEYS } from '@/features/tickets/constants'

import { TicketDetailSheet } from './components/ticket-detail-sheet'
import { TicketSearchBar } from './components/ticket-search-bar'
import { TicketStatusTabs } from './components/ticket-status-tabs'
import { TicketsTable } from './components/tickets-table'
import {
  buildAdminTicketQuery,
  TICKET_PAGE_SIZE,
  type TicketManagementSearch,
} from './lib/ticket-management'

const route = getRouteApi('/_authenticated/ticket-management/')

/** Support console: filter and search tickets, then answer one in a sheet. */
export function TicketManagement() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const [openTicketId, setOpenTicketId] = useState<number | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)

  const query = buildAdminTicketQuery(search)
  const listQuery = useQuery({
    queryKey: [...TICKET_QUERY_KEYS.adminList, query],
    queryFn: () => getAdminTickets(query),
    placeholderData: keepPreviousData,
  })
  const statsQuery = useQuery({
    queryKey: TICKET_QUERY_KEYS.adminStats,
    queryFn: getAdminTicketStats,
  })

  const updateSearch = (patch: TicketManagementSearch) => {
    void navigate({ search: (previous) => ({ ...previous, ...patch }) })
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Ticket Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <TicketStatusTabs
                value={search.status}
                stats={statsQuery.data}
                onChange={(status) => updateSearch({ status, page: 1 })}
              />
              <TicketSearchBar
                key={search.keyword ?? ''}
                keyword={search.keyword ?? ''}
                onSearch={(keyword) =>
                  updateSearch({ keyword: keyword || undefined, page: 1 })
                }
              />
            </div>

            {listQuery.isError && (
              <Alert variant='destructive'>
                <AlertDescription className='flex flex-wrap items-center gap-2'>
                  <span>{listQuery.error.message}</span>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void listQuery.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </AlertDescription>
              </Alert>
            )}

            {listQuery.isPending && (
              <div className='space-y-2'>
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
                <Skeleton className='h-10 w-full' />
              </div>
            )}

            {listQuery.data && (
              <TicketsTable
                tickets={listQuery.data.items}
                page={query.p}
                pageSize={TICKET_PAGE_SIZE}
                total={listQuery.data.total}
                onPageChange={(page) => updateSearch({ page })}
                onOpen={(ticket) => {
                  setOpenTicketId(ticket.id)
                  setSheetOpen(true)
                }}
              />
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <TicketDetailSheet
        ticketId={openTicketId}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
      />
    </>
  )
}
