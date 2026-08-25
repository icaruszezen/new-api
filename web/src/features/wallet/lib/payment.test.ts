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

import { PAYMENT_TYPES } from '../constants'
import {
  dispatchSelectedPayment,
  getPaymentErrorMessage,
  getPaymentMethodKey,
  isStripePayment,
  isWaffoPayment,
  isWaffoPancakePayment,
} from './payment'

describe('payment type classification', () => {
  test('keeps Waffo and Waffo Pancake on their dedicated flows', () => {
    expect(isWaffoPayment(PAYMENT_TYPES.WAFFO)).toBe(true)
    expect(isWaffoPayment(PAYMENT_TYPES.WAFFO_PANCAKE)).toBe(false)
    expect(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO_PANCAKE)).toBe(true)
    expect(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO)).toBe(false)
    expect(isStripePayment(PAYMENT_TYPES.STRIPE)).toBe(true)
  })
})

describe('payment dispatch', () => {
  test('keeps the selected Waffo method index through confirmation', async () => {
    const calls: string[] = []
    const success = await dispatchSelectedPayment(
      { name: 'Waffo Card', type: PAYMENT_TYPES.WAFFO },
      120,
      3,
      {
        regular: async () => {
          calls.push('regular')
          return false
        },
        waffo: async (amount, index) => {
          calls.push(`waffo:${amount}:${index}`)
          return true
        },
        waffoPancake: async () => {
          calls.push('pancake')
          return false
        },
      }
    )

    expect(success).toBe(true)
    expect(calls).toEqual(['waffo:120:3'])
  })

  test('does not create a Waffo order without a selected method index', async () => {
    let called = false
    const success = await dispatchSelectedPayment(
      { name: 'Waffo Card', type: PAYMENT_TYPES.WAFFO },
      120,
      null,
      {
        regular: async () => false,
        waffo: async () => {
          called = true
          return true
        },
        waffoPancake: async () => false,
      }
    )

    expect(success).toBe(false)
    expect(called).toBe(false)
  })

  test('charges an epay method through the gateway it is bound to', async () => {
    const calls: string[] = []
    const success = await dispatchSelectedPayment(
      { name: 'Alipay Backup', type: 'alipay', gateway_id: 'gw_backup' },
      50,
      null,
      {
        regular: async (amount, paymentType, gatewayId) => {
          calls.push(`regular:${amount}:${paymentType}:${gatewayId}`)
          return true
        },
        waffo: async () => false,
        waffoPancake: async () => false,
      }
    )

    expect(success).toBe(true)
    expect(calls).toEqual(['regular:50:alipay:gw_backup'])
  })

  test('leaves the gateway unset for a method on the default gateway', async () => {
    const calls: (string | undefined)[] = []
    await dispatchSelectedPayment(
      { name: 'Alipay', type: 'alipay' },
      50,
      null,
      {
        regular: async (_amount, _paymentType, gatewayId) => {
          calls.push(gatewayId)
          return true
        },
        waffo: async () => false,
        waffoPancake: async () => false,
      }
    )

    expect(calls).toEqual([undefined])
  })
})

describe('payment method identity', () => {
  test('separates the same epay type configured on different gateways', () => {
    const defaultGatewayMethod = { name: 'Alipay', type: 'alipay' }
    const backupGatewayMethod = {
      name: 'Alipay Backup',
      type: 'alipay',
      gateway_id: 'gw_backup',
    }

    expect(getPaymentMethodKey(defaultGatewayMethod)).toBe(
      getPaymentMethodKey({ name: 'Alipay renamed', type: 'alipay' })
    )
    expect(getPaymentMethodKey(defaultGatewayMethod)).not.toBe(
      getPaymentMethodKey(backupGatewayMethod)
    )
  })
})

describe('payment error message', () => {
  test('prefers the backend data reason over a generic error message', () => {
    expect(getPaymentErrorMessage('error', '拉起支付失败')).toBe('拉起支付失败')
  })

  test('falls back to the response message when data is not a reason string', () => {
    expect(getPaymentErrorMessage('Payment request failed', { url: '' })).toBe(
      'Payment request failed'
    )
    expect(getPaymentErrorMessage('error', '   ')).toBe('error')
  })

  test('falls back to the default copy when both message and data are empty', () => {
    expect(getPaymentErrorMessage(undefined, undefined)).toBe(
      'Payment request failed'
    )
  })
})
