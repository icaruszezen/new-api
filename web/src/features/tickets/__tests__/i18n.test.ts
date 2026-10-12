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
import { describe, expect, test } from 'vitest'

import { TICKET_STATUS_CONFIG } from '@/features/tickets/constants'
import zhTW from '@/i18n/locales/zh-TW.json'
import zh from '@/i18n/locales/zh.json'

const TICKET_KEYS = [
  // Statuses and audit log labels
  'Awaiting support',
  'In progress',
  'Resolved',
  'Closed',
  'Replied to ticket #{{id}}',
  'Changed status of ticket #{{id}}',
  // Shared thread, picker and composer
  'Support',
  'You',
  'Attachment {{index}}',
  'Attachment {{index}} could not be loaded',
  'View attachment {{index}}',
  'Add images',
  'Remove image',
  '{{count}}/{{max}} images',
  'Only PNG, JPEG, GIF and WebP images are supported',
  'Each image must be {{max}} MB or smaller',
  'You can attach up to {{max}} images',
  'Reply',
  'Write a reply...',
  'Send reply',
  'Too many requests. Please try again later.',
  'The upload is too large. Try fewer or smaller images.',
  // User pages and floating button
  'Support tickets',
  'Unread updates: {{count}}',
  'Report a problem or ask a question. Our support team replies here, and you can follow up at any time.',
  'New ticket',
  'Filter by status',
  'All tickets',
  'New update',
  'Updated {{time}}',
  'Created {{time}}',
  'Pagination',
  'Failed to load your tickets',
  'No tickets yet',
  'Describe your problem and attach screenshots if they help. We will reply on this page.',
  'No tickets on this page.',
  'No tickets with this status.',
  'Tell us what happened. Screenshots help us find the problem faster.',
  'Subject',
  'Summarize the problem in one line',
  'Issue description',
  'What happened, when, and what did you expect? Include error messages if you have them.',
  'Images',
  'Optional. Up to {{images}} PNG, JPEG, GIF or WebP images, {{size}} MB each. You can also paste or drop them here.',
  'Submit ticket',
  'Ticket submitted',
  'Please enter a subject',
  'Please describe your issue',
  'The subject can be at most {{max}} characters',
  'The description can be at most {{max}} characters',
  'Conversation',
  'Close ticket',
  'Close this ticket?',
  'Close it once your problem is solved. You will not be able to reply afterwards, but you can always submit a new ticket.',
  'Ticket closed',
  'This ticket is closed. Submit a new ticket if you still need help.',
  'Failed to load this ticket',
  'Check your connection and try again.',
  'Ticket not found',
  'This ticket does not exist or belongs to another account.',
  // Admin console
  'Ticket Management',
  'Search tickets',
  'Ticket ID, subject or username',
  'Messages',
  'Unread',
  'No tickets found',
  '{{count}} ticket(s)',
  '{{count}} unread',
  'Submitted by',
  'Status after reply',
  'Keep automatic',
  'Ticket status updated',
  'Failed to update ticket status',
  'Reply sent',
  'Failed to send reply',
] as const

const CHINESE_LOCALES = [
  ['zh', zh.translation],
  ['zh-TW', zhTW.translation],
] as const

describe('support ticket Chinese locales', () => {
  test('zh and zh-TW translate every ticket key instead of falling back to English keys', () => {
    for (const [locale, translation] of CHINESE_LOCALES) {
      for (const key of TICKET_KEYS) {
        expect(translation[key], `${locale} is missing "${key}"`).toEqual(
          expect.any(String)
        )
        expect(translation[key], `${locale} left "${key}" in English`).not.toBe(
          key
        )
      }
    }
  })

  test('every ticket status label key is covered by the translated keys', () => {
    const translatedKeys: readonly string[] = TICKET_KEYS

    for (const config of Object.values(TICKET_STATUS_CONFIG)) {
      expect(translatedKeys).toContain(config.labelKey)
    }
  })

  test('interpolated ticket messages keep their placeholders in every Chinese locale', () => {
    const placeholderPattern = /\{\{\s*(\w+)\s*\}\}/g

    for (const key of TICKET_KEYS.filter((entry) => entry.includes('{{'))) {
      const expected = [...key.matchAll(placeholderPattern)]
        .map((match) => match[1])
        .sort()

      for (const [locale, translation] of CHINESE_LOCALES) {
        const actual = [...translation[key].matchAll(placeholderPattern)]
          .map((match) => match[1])
          .sort()
        expect(
          actual,
          `${locale} changed the placeholders of "${key}"`
        ).toEqual(expected)
      }
    }
  })

  test('zh and zh-TW use the agreed ticket status names', () => {
    expect(
      Object.values(TICKET_STATUS_CONFIG).map(
        (config) =>
          zh.translation[config.labelKey as keyof typeof zh.translation]
      )
    ).toEqual(['待处理', '处理中', '已解决', '已关闭'])
    expect(
      Object.values(TICKET_STATUS_CONFIG).map(
        (config) =>
          zhTW.translation[config.labelKey as keyof typeof zhTW.translation]
      )
    ).toEqual(['待處理', '處理中', '已解決', '已關閉'])
  })
})
