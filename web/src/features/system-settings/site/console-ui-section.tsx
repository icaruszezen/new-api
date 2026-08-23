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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

// react-hook-form reads dots as nested paths, so the form mirrors the option
// namespace and is flattened back to `console_setting.ui_skin` on submit.
const consoleUiSchema = z.object({
  console_setting: z.object({
    ui_skin: z.enum(['classic', 'next']),
  }),
})

type ConsoleUiFormValues = z.infer<typeof consoleUiSchema>

type ConsoleUiSectionProps = {
  defaultValues: ConsoleUiFormValues
}

export function ConsoleUiSection(props: ConsoleUiSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<ConsoleUiFormValues>({
    resolver: zodResolver(consoleUiSchema),
    defaultValues: props.defaultValues,
  })

  useResetForm(form, props.defaultValues)

  const onSubmit = async (values: ConsoleUiFormValues) => {
    if (
      values.console_setting.ui_skin ===
      props.defaultValues.console_setting.ui_skin
    ) {
      toast.info(t('No changes to save'))
      return
    }

    await updateOption.mutateAsync({
      key: 'console_setting.ui_skin',
      value: values.console_setting.ui_skin,
    })
  }

  const skinOptions = [
    { value: 'classic', label: t('Current interface') },
    { value: 'next', label: t('Next interface') },
  ]

  return (
    <SettingsSection title={t('Console interface')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save console interface'
          />
          <FormField
            control={form.control}
            name='console_setting.ui_skin'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Console interface')}</FormLabel>
                <Select
                  items={skinOptions}
                  onValueChange={field.onChange}
                  value={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value='classic'>
                        {t('Current interface')}
                      </SelectItem>
                      <SelectItem value='next'>
                        {t('Next interface')}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {t(
                    'Choose the console interface for all users. This is a site-wide setting and users cannot override it.'
                  )}
                </FormDescription>
                <FormDescription>
                  {t(
                    'The next interface currently matches the current interface. Switching now only reserves this setting.'
                  )}
                </FormDescription>
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
