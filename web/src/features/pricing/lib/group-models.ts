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
import type { PricingModel } from '../types'

/**
 * Collapse same-named pricing rows into one catalog entry.
 *
 * `/api/pricing` already keys by `model_name`; this keeps the list contract
 * if a response ever repeats a name, by unioning `enable_groups`.
 */
export function groupModelsByName(models: PricingModel[]): PricingModel[] {
  if (models.length === 0) {
    return []
  }

  const grouped = new Map<string, PricingModel>()

  for (const model of models) {
    const name = model.model_name
    const existing = grouped.get(name)
    if (!existing) {
      grouped.set(name, {
        ...model,
        enable_groups: [...(model.enable_groups || [])],
      })
      continue
    }

    const groups = new Set(existing.enable_groups)
    for (const group of model.enable_groups || []) {
      groups.add(group)
    }
    existing.enable_groups = [...groups]
  }

  return [...grouped.values()]
}
