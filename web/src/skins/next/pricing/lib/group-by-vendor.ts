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
import type { PricingModel } from '@/features/pricing/types'

export const OTHER_VENDOR_GROUP_ID = 'other'

export type VendorModelGroup = {
  id: string
  name: string
  icon?: string
  models: PricingModel[]
}

function vendorGroupKey(model: PricingModel): string {
  return model.vendor_name?.trim() || OTHER_VENDOR_GROUP_ID
}

const FEATURED_VENDOR_ORDER = ['openai', 'anthropic', 'gemini', 'grok'] as const

function vendorSortKey(group: VendorModelGroup): number {
  const name = group.name.trim().toLowerCase()
  const rank = FEATURED_VENDOR_ORDER.findIndex((item) => item === name)
  return rank === -1 ? FEATURED_VENDOR_ORDER.length : rank
}

function compareVendorGroups(a: VendorModelGroup, b: VendorModelGroup): number {
  const rankDelta = vendorSortKey(a) - vendorSortKey(b)
  if (rankDelta !== 0) return rankDelta
  return a.name.localeCompare(b.name)
}

export function groupModelsByVendor(models: PricingModel[]): VendorModelGroup[] {
  if (models.length === 0) {
    return []
  }

  const groups = new Map<string, VendorModelGroup>()

  for (const model of models) {
    const id = vendorGroupKey(model)
    const existing = groups.get(id)
    if (existing) {
      existing.models.push(model)
      if (!existing.icon && model.vendor_icon) {
        existing.icon = model.vendor_icon
      }
      continue
    }

    groups.set(id, {
      id,
      name: model.vendor_name?.trim() || '',
      icon: model.vendor_icon,
      models: [model],
    })
  }

  const named = [...groups.values()].filter(
    (group) => group.id !== OTHER_VENDOR_GROUP_ID
  )
  named.sort(compareVendorGroups)

  const other = groups.get(OTHER_VENDOR_GROUP_ID)
  if (other) {
    return [...named, other]
  }
  return named
}
