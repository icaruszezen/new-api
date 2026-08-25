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
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Button } from '@/components/ui/button'

import { EpayGatewayDialog, type EpayGatewayData } from './epay-gateway-dialog'

type EpayGatewaysVisualEditorProps = {
  gateways: EpayGatewayData[]
  onGatewaysChange: (gateways: EpayGatewayData[]) => void
  notifyUrlPreviewBase?: string
}

export function EpayGatewaysVisualEditor(props: EpayGatewaysVisualEditorProps) {
  const { t } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editData, setEditData] = useState<EpayGatewayData | null>(null)

  const handleSave = (data: EpayGatewayData) => {
    const index = props.gateways.findIndex(
      (gateway) => gateway.id === (editData?.id ?? data.id)
    )
    if (index === -1) {
      props.onGatewaysChange([...props.gateways, data])
      return
    }
    const updated = [...props.gateways]
    updated[index] = data
    props.onGatewaysChange(updated)
  }

  const handleDelete = (gateway: EpayGatewayData) => {
    props.onGatewaysChange(
      props.gateways.filter((item) => item.id !== gateway.id)
    )
  }

  const handleEdit = (gateway: EpayGatewayData) => {
    setEditData(gateway)
    setDialogOpen(true)
  }

  const handleAdd = () => {
    setEditData(null)
    setDialogOpen(true)
  }

  return (
    <div className='space-y-4'>
      <div className='flex justify-end'>
        <Button
          type='button'
          onClick={(e) => {
            e.preventDefault()
            e.stopPropagation()
            handleAdd()
          }}
          className='w-full sm:w-auto'
        >
          <Plus className='h-4 w-4 sm:mr-2' />
          <span className='sm:inline'>{t('Add gateway')}</span>
        </Button>
      </div>

      {props.gateways.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
          {t(
            'No additional gateways configured. Payment methods without a gateway use the default gateway above.'
          )}
        </div>
      ) : (
        <div className='rounded-md border'>
          {/* Desktop table view */}
          <StaticDataTable
            className='hidden rounded-none border-0 md:block'
            data={props.gateways}
            getRowKey={(gateway) => gateway.id}
            columns={[
              {
                id: 'name',
                header: t('Name'),
                cellClassName: 'font-medium',
                cell: (gateway) => gateway.name,
              },
              {
                id: 'id',
                header: t('Gateway ID'),
                cell: (gateway) => (
                  <code className='bg-muted rounded px-1.5 py-0.5 text-sm'>
                    {gateway.id}
                  </code>
                ),
              },
              {
                id: 'endpoint',
                header: t('Epay endpoint'),
                cell: (gateway) => (
                  <span className='text-muted-foreground truncate text-sm'>
                    {gateway.pay_address}
                  </span>
                ),
              },
              {
                id: 'merchant',
                header: t('Epay merchant ID'),
                cell: (gateway) => (
                  <span className='font-mono text-sm'>{gateway.epay_id}</span>
                ),
              },
              {
                id: 'actions',
                header: t('Actions'),
                className: 'text-right',
                cellClassName: 'text-right',
                cell: (gateway) => (
                  <StaticRowActions
                    editLabel={t('Edit')}
                    deleteLabel={t('Delete')}
                    menuLabel={t('Open menu')}
                    onEdit={() => handleEdit(gateway)}
                    onDelete={() => handleDelete(gateway)}
                  />
                ),
              },
            ]}
          />

          {/* Mobile card view */}
          <div className='divide-y md:hidden'>
            {props.gateways.map((gateway) => (
              <div key={gateway.id} className='p-4'>
                <div className='mb-3 flex items-start justify-between'>
                  <div className='min-w-0 flex-1'>
                    <div className='mb-1 font-medium'>{gateway.name}</div>
                    <code className='bg-muted rounded px-1.5 py-0.5 text-xs'>
                      {gateway.id}
                    </code>
                  </div>
                  <div className='flex gap-1'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        handleEdit(gateway)
                      }}
                    >
                      <Pencil className='h-4 w-4' />
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        handleDelete(gateway)
                      }}
                    >
                      <Trash2 className='h-4 w-4' />
                    </Button>
                  </div>
                </div>
                <div className='space-y-2 text-sm'>
                  <div className='flex items-center gap-2'>
                    <span className='text-muted-foreground min-w-24'>
                      {t('Epay endpoint')}
                    </span>
                    <span className='text-muted-foreground truncate'>
                      {gateway.pay_address}
                    </span>
                  </div>
                  <div className='flex items-center gap-2'>
                    <span className='text-muted-foreground min-w-24'>
                      {t('Epay merchant ID')}
                    </span>
                    <span className='font-mono'>{gateway.epay_id}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <EpayGatewayDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSave={handleSave}
        editData={editData}
        notifyUrlPreviewBase={props.notifyUrlPreviewBase}
      />
    </div>
  )
}
