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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import type { ApiInfoItem } from '@/features/dashboard/types'
import { useApiKeys } from '@/features/keys/components/api-keys-provider'
import type { ApiKey } from '@/features/keys/types'
import { getUserModels } from '@/lib/api'
import { copyToClipboard } from '@/lib/copy-to-clipboard'

import { streamChatCompletion } from '../lib/test-connection-stream'
import {
  buildCurlCommand,
  buildEndpointOptions,
  buildTestUrl,
  formatsForModel,
  getPreferredApiKey,
  parseModelLimits,
  pickFormat,
  pickModel,
  resolveFallbackOrigin,
  type TestConnectionPhase,
} from '../lib/test-connection'

export function useTestConnection(args: {
  open: boolean
  keys: ApiKey[]
  apiInfoItems: ApiInfoItem[]
  serverAddress?: string
}) {
  const { t } = useTranslation()
  const { resolveRealKey } = useApiKeys()
  const abortRef = useRef<AbortController | null>(null)

  const fallbackOrigin = resolveFallbackOrigin(args.serverAddress)
  const endpoints = useMemo(
    () => buildEndpointOptions(args.apiInfoItems, fallbackOrigin, t('Default')),
    [args.apiInfoItems, fallbackOrigin, t]
  )

  const [selectedKeyId, setSelectedKeyId] = useState<number | null>(null)
  const [selectedEndpointId, setSelectedEndpointId] = useState<string | null>(
    null
  )
  const [modelOverride, setModelOverride] = useState('')
  const [formatOverride, setFormatOverride] = useState('')
  const [phase, setPhase] = useState<TestConnectionPhase>('idle')
  const [responseText, setResponseText] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (!args.open) {
      abortRef.current?.abort()
      abortRef.current = null
      setPhase('idle')
      setResponseText('')
      setErrorMessage('')
      setSelectedKeyId(null)
      setSelectedEndpointId(null)
      setModelOverride('')
      setFormatOverride('')
      return
    }

    setSelectedKeyId((current) => {
      if (current && args.keys.some((key) => key.id === current)) return current
      return getPreferredApiKey(args.keys)?.id ?? null
    })
    setSelectedEndpointId((current) => {
      if (current && endpoints.some((item) => item.id === current)) {
        return current
      }
      return endpoints[0]?.id ?? null
    })
  }, [args.open, args.keys, endpoints])

  useEffect(() => {
    setModelOverride('')
    setFormatOverride('')
  }, [selectedKeyId])

  const selectedKey =
    args.keys.find((key) => key.id === selectedKeyId) ?? null
  const selectedEndpoint =
    endpoints.find((item) => item.id === selectedEndpointId) ?? null
  const limitedModels = selectedKey ? parseModelLimits(selectedKey) : null

  const modelsQuery = useQuery({
    queryKey: [
      'console',
      'test-connection',
      'models',
      selectedKey?.id,
      selectedKey?.group,
    ],
    queryFn: async () => {
      const result = await getUserModels(selectedKey?.group || undefined)
      return result.success ? (result.data ?? []) : []
    },
    enabled: args.open && Boolean(selectedKey) && limitedModels === null,
  })

  const models = limitedModels ?? modelsQuery.data ?? []
  const selectedModel = pickModel(models, modelOverride)
  const formats = formatsForModel(selectedModel)
  const selectedFormat = pickFormat(formats, formatOverride)
  const canSubmit = Boolean(selectedKey && selectedEndpoint && selectedModel)
  const isModelsLoading =
    limitedModels === null && Boolean(selectedKey) && modelsQuery.isFetching

  const startTest = async () => {
    if (!selectedKey || !selectedEndpoint || !selectedModel) return

    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setPhase('running')
    setResponseText('')
    setErrorMessage('')

    try {
      const realKey = await resolveRealKey(selectedKey.id)
      if (!realKey) {
        if (controller.signal.aborted) return
        setPhase('error')
        setErrorMessage(t('An unexpected error occurred'))
        return
      }
      if (controller.signal.aborted) return

      await streamChatCompletion({
        url: buildTestUrl(selectedEndpoint.baseUrl, selectedFormat),
        apiKey: realKey,
        model: selectedModel,
        format: selectedFormat,
        signal: controller.signal,
        onContent: (chunk) => {
          setResponseText((current) => current + chunk)
        },
      })
      if (!controller.signal.aborted) setPhase('success')
    } catch (error) {
      if (controller.signal.aborted) return
      setPhase('error')
      setErrorMessage(
        error instanceof Error ? error.message : t('An unexpected error occurred')
      )
    }
  }

  const copyTestLink = async () => {
    if (!selectedKey || !selectedEndpoint || !selectedModel) return
    const realKey = await resolveRealKey(selectedKey.id)
    if (!realKey) return
    const copied = await copyToClipboard(
      buildCurlCommand({
        endpoint: buildTestUrl(selectedEndpoint.baseUrl, selectedFormat),
        apiKey: realKey,
        model: selectedModel,
        format: selectedFormat,
      })
    )
    if (copied) {
      toast.success(t('Copied to clipboard'))
      return
    }
    toast.error(t('An unexpected error occurred'))
  }

  return {
    canSubmit,
    copyTestLink,
    endpoints,
    errorMessage,
    formats,
    isModelsLoading,
    models,
    phase,
    responseText,
    selectedEndpointId,
    selectedFormat,
    selectedKeyId,
    selectedModel,
    setFormatOverride,
    setModelOverride,
    setSelectedEndpointId,
    setSelectedKeyId,
    startTest,
  }
}
