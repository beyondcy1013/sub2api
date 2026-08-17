import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CronScheduleBuilder from '../CronScheduleBuilder.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key
  })
}))

describe('CronScheduleBuilder', () => {
  it('recognizes an interval expression and changes it through mouse controls', async () => {
    const wrapper = mount(CronScheduleBuilder, {
      props: {
        modelValue: '*/30 * * * *',
        'onUpdate:modelValue': (value: string) => wrapper.setProps({ modelValue: value })
      }
    })

    expect(wrapper.get('[data-testid="cron-mode-interval"]').classes()).toContain('bg-primary-500')
    await wrapper.get('[data-testid="cron-interval"]').setValue('10')
    expect(wrapper.props('modelValue')).toBe('*/10 * * * *')
  })

  it('builds daily and weekly expressions without typing cron syntax', async () => {
    const wrapper = mount(CronScheduleBuilder, {
      props: {
        modelValue: '*/30 * * * *',
        'onUpdate:modelValue': (value: string) => wrapper.setProps({ modelValue: value })
      }
    })

    await wrapper.get('[data-testid="cron-mode-daily"]').trigger('click')
    await wrapper.get('[data-testid="cron-hour"]').setValue('14')
    await wrapper.get('[data-testid="cron-minute"]').setValue('25')
    expect(wrapper.props('modelValue')).toBe('25 14 * * *')

    await wrapper.get('[data-testid="cron-mode-weekly"]').trigger('click')
    await wrapper.get('[data-testid="cron-weekday"]').setValue('5')
    expect(wrapper.props('modelValue')).toBe('25 14 * * 5')
  })

  it('keeps unsupported expressions available in custom mode', async () => {
    const wrapper = mount(CronScheduleBuilder, {
      props: { modelValue: '0 9 1 * *' }
    })

    expect(wrapper.get('[data-testid="cron-mode-custom"]').classes()).toContain('bg-primary-500')
    expect(wrapper.props('modelValue')).toBe('0 9 1 * *')
  })

  it('expands cron help through the help button', async () => {
    const wrapper = mount(CronScheduleBuilder, {
      props: { modelValue: '*/30 * * * *' }
    })

    const helpButton = wrapper.get('[data-testid="cron-builder-help"]')
    expect(helpButton.attributes('aria-expanded')).toBe('false')
    expect(wrapper.text()).not.toContain('admin.scheduledTests.builderHelpBody')

    await helpButton.trigger('click')

    expect(helpButton.attributes('aria-expanded')).toBe('true')
    expect(wrapper.text()).toContain('admin.scheduledTests.builderHelpBody')
    expect(wrapper.text()).toContain('admin.scheduledTests.cronFieldOrder')
  })

  it('is present in both add and edit scheduled-test forms', () => {
    const panelSource = readFileSync(resolve(__dirname, '../ScheduledTestsPanel.vue'), 'utf8')
    const builders = panelSource.match(/<CronScheduleBuilder\s+v-model=/g) || []

    expect(builders).toHaveLength(2)
    expect(panelSource).toContain('<CronScheduleBuilder v-model="newPlan.cron_expression" />')
    expect(panelSource).toContain('<CronScheduleBuilder v-model="editForm.cron_expression" />')
  })
})
