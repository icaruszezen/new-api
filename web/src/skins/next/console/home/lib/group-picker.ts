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
import type { PricingModel, PricingVendor } from '@/features/pricing/types'

export const GROUP_PICKER_AUTO_SECTION_ID = 'auto'
export const GROUP_PICKER_OTHER_SECTION_ID = 'other'
export const GROUP_PICKER_FLAT_SECTION_ID = 'flat'

export type GroupPickerOption = {
  value: string
  label: string
  desc?: string
  ratio?: number | string
}

export type GroupPickerSection = {
  id: string
  title: string
  icon?: string
  groups: GroupPickerOption[]
}

export type NormalizedVendor = {
  id: string
  title: string
  icon?: string
}

const FEATURED_VENDOR_IDS = ['openai', 'claude', 'gemini', 'grok'] as const

type FeaturedVendorId = (typeof FEATURED_VENDOR_IDS)[number]

const FEATURED_VENDOR_META: Record<
  FeaturedVendorId,
  { title: string; icon: string }
> = {
  openai: { title: 'OpenAI', icon: 'OpenAI' },
  claude: { title: 'Claude', icon: 'Claude' },
  gemini: { title: 'Gemini', icon: 'Gemini' },
  grok: { title: 'Grok', icon: 'Grok' },
}

export function normalizeVendor(
  name?: string,
  icon?: string
): NormalizedVendor {
  const trimmedName = name?.trim() ?? ''
  const trimmedIcon = icon?.trim() ?? ''
  const hay = `${trimmedName} ${trimmedIcon}`.toLowerCase()
  const featuredId = featuredVendorIdFromHay(hay)
  if (featuredId) {
    const meta = FEATURED_VENDOR_META[featuredId]
    return {
      id: featuredId,
      title: meta.title,
      icon: trimmedIcon || meta.icon,
    }
  }
  if (!trimmedName) {
    return {
      id: GROUP_PICKER_OTHER_SECTION_ID,
      title: '',
      icon: trimmedIcon || undefined,
    }
  }
  return {
    id: trimmedName,
    title: trimmedName,
    icon: trimmedIcon || undefined,
  }
}

export function resolvePricingModels(
  models: PricingModel[],
  vendors: PricingVendor[]
): PricingModel[] {
  if (vendors.length === 0) return models
  const vendorMap = new Map(vendors.map((vendor) => [vendor.id, vendor]))
  return models.map((model) => {
    const vendor =
      model.vendor_id != null ? vendorMap.get(model.vendor_id) : undefined
    if (!vendor) return model
    return {
      ...model,
      vendor_name: vendor.name,
      vendor_icon: vendor.icon,
      vendor_description: vendor.description,
    }
  })
}

export function sortGroupsByRatio(
  groups: GroupPickerOption[]
): GroupPickerOption[] {
  return [...groups].sort(compareGroupsByRatio)
}

export function groupUserGroupsByVendor(
  groups: GroupPickerOption[],
  models: Array<
    Pick<PricingModel, 'enable_groups' | 'vendor_name' | 'vendor_icon'>
  >
): GroupPickerSection[] {
  const sections = new Map<string, GroupPickerSection>()
  const autoGroups: GroupPickerOption[] = []

  for (const group of groups) {
    if (group.value === 'auto') {
      autoGroups.push(group)
      continue
    }

    const vendorsForGroup = new Map<string, NormalizedVendor>()
    for (const model of models) {
      if (!model.enable_groups.includes(group.value)) continue
      const vendor = normalizeVendor(model.vendor_name, model.vendor_icon)
      vendorsForGroup.set(vendor.id, vendor)
    }

    if (vendorsForGroup.size === 0) {
      upsertSection(
        sections,
        {
          id: GROUP_PICKER_OTHER_SECTION_ID,
          title: '',
        },
        group
      )
      continue
    }

    for (const vendor of vendorsForGroup.values()) {
      upsertSection(sections, vendor, group)
    }
  }

  for (const section of sections.values()) {
    section.groups.sort(compareGroupsByRatio)
  }

  const named = [...sections.values()].filter(
    (section) => section.id !== GROUP_PICKER_OTHER_SECTION_ID
  )
  named.sort(compareVendorSections)

  const result: GroupPickerSection[] = []
  if (autoGroups.length > 0) {
    result.push({
      id: GROUP_PICKER_AUTO_SECTION_ID,
      title: '',
      groups: autoGroups,
    })
  }
  result.push(...named)
  const other = sections.get(GROUP_PICKER_OTHER_SECTION_ID)
  if (other) result.push(other)
  return result
}

export function filterGroupPickerSections(
  sections: GroupPickerSection[],
  search: string
): GroupPickerSection[] {
  const query = search.trim().toLowerCase()
  if (!query) return sections

  return sections.flatMap((section) => {
    const groups = section.groups.filter((group) => {
      const ratioText = String(group.ratio ?? '').toLowerCase()
      return (
        group.value.toLowerCase().includes(query) ||
        group.label.toLowerCase().includes(query) ||
        group.desc?.toLowerCase().includes(query) ||
        ratioText.includes(query) ||
        section.title.toLowerCase().includes(query)
      )
    })
    if (groups.length === 0) return []
    return [{ ...section, groups }]
  })
}

function featuredVendorIdFromHay(hay: string): FeaturedVendorId | null {
  if (hay.includes('openai')) return 'openai'
  if (hay.includes('claude') || hay.includes('anthropic')) return 'claude'
  if (hay.includes('gemini')) return 'gemini'
  if (hay.includes('grok')) return 'grok'
  return null
}

function numericRatio(ratio?: number | string): number | null {
  if (typeof ratio !== 'number' || !Number.isFinite(ratio)) return null
  return ratio
}

function compareGroupsByRatio(
  left: GroupPickerOption,
  right: GroupPickerOption
): number {
  const leftRatio = numericRatio(left.ratio)
  const rightRatio = numericRatio(right.ratio)
  if (leftRatio === null && rightRatio === null) {
    return left.value.localeCompare(right.value)
  }
  if (leftRatio === null) return 1
  if (rightRatio === null) return -1
  if (leftRatio !== rightRatio) return leftRatio - rightRatio
  return left.value.localeCompare(right.value)
}

function featuredVendorRank(sectionId: string): number {
  const rank = FEATURED_VENDOR_IDS.indexOf(sectionId as FeaturedVendorId)
  return rank === -1 ? FEATURED_VENDOR_IDS.length : rank
}

function compareVendorSections(
  left: GroupPickerSection,
  right: GroupPickerSection
): number {
  const rankDelta = featuredVendorRank(left.id) - featuredVendorRank(right.id)
  if (rankDelta !== 0) return rankDelta
  return left.title.localeCompare(right.title)
}

function upsertSection(
  sections: Map<string, GroupPickerSection>,
  vendor: NormalizedVendor,
  group: GroupPickerOption
): void {
  const existing = sections.get(vendor.id)
  if (!existing) {
    sections.set(vendor.id, {
      id: vendor.id,
      title: vendor.title,
      icon: vendor.icon,
      groups: [group],
    })
    return
  }
  if (!existing.icon && vendor.icon) {
    existing.icon = vendor.icon
  }
  if (existing.groups.some((item) => item.value === group.value)) return
  existing.groups.push(group)
}
