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
import { ConsoleKeyManagement } from './components/key-management'
import { ConsoleShortcutRow } from './components/shortcut-row'
import { ConsoleStatsRow } from './components/stats-row'

export function NextConsoleHome() {
  return (
    <div className='flex flex-col gap-5 pb-10'>
      <div className='grid grid-cols-1 gap-3 lg:grid-cols-12 lg:items-stretch'>
        <div className='min-w-0 h-full lg:col-span-9'>
          <ConsoleStatsRow />
        </div>
        <div className='lg:col-span-3'>
          <ConsoleShortcutRow />
        </div>
      </div>
      <ConsoleKeyManagement />
    </div>
  )
}
