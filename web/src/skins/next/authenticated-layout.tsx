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
// Imported from the concrete module instead of the `@/components/layout`
// barrel, which reaches the sidebar config and cycles back into features.
import { AuthenticatedLayout } from '@/components/layout/components/authenticated-layout'

/**
 * Next console shell. It intentionally renders the existing layout so this
 * release stays identical to `classic`; the future shell replaces the body of
 * this component instead of forking the layout tree elsewhere.
 */
export function NextAuthenticatedLayout() {
  return <AuthenticatedLayout />
}
