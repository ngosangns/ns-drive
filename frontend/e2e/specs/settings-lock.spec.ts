import { after, before, describe, it } from 'node:test'
import assert from 'node:assert/strict'
import type { Page } from 'puppeteer'
import {
  clickTestId,
  closeBrowser,
  closePage,
  collectCoverage,
  newPage,
  typeTestId,
  waitForTestId,
} from '../helpers/browser.js'
import { ensureSession, unlock } from '../helpers/auth.js'
import { loadRuntime } from '../helpers/env.js'

describe('settings and lock', () => {
  let page: Page

  before(async () => {
    page = await newPage()
    await unlock(page)
  })

  after(async () => {
    if (page) await closePage(page)
    await closeBrowser()
  })

  it('navigates to settings and back', async () => {
    await clickTestId(page, 'nav-settings')
    await waitForTestId(page, 'page-settings')
    const h1 = await page.$eval('h1', (el) => el.textContent?.trim() ?? '')
    assert.equal(h1, 'Settings')
    await clickTestId(page, 'nav-workspace')
    await waitForTestId(page, 'page-workspace')
    await collectCoverage(page)
  })

  it('rejects a short password then rotates and restores the master password', async () => {
    await ensureSession(page)
    await clickTestId(page, 'nav-settings')
    await waitForTestId(page, 'page-settings')
    await clickTestId(page, 'theme-light')
    await clickTestId(page, 'theme-dark')

    const { password } = loadRuntime()
    await typeTestId(page, 'settings-old-password', password)
    await typeTestId(page, 'settings-new-password', 'ab')
    await clickTestId(page, 'settings-change-password')
    await page.waitForFunction(
      () => {
        const msg = document.querySelector('[data-testid="settings-msg"]')
        return !!msg && (msg.textContent?.includes('at least 4') ?? false)
      },
      { timeout: 8_000 },
    )

    const tempPwd = `${password}-tmp`
    await typeTestId(page, 'settings-old-password', password)
    await typeTestId(page, 'settings-new-password', tempPwd)
    await clickTestId(page, 'settings-change-password')
    await waitForTestId(page, 'page-unlock', 10_000)
    await typeTestId(page, 'unlock-password', tempPwd)
    await clickTestId(page, 'unlock-submit')
    await waitForTestId(page, 'page-workspace')

    await clickTestId(page, 'nav-settings')
    await waitForTestId(page, 'page-settings')
    await typeTestId(page, 'settings-old-password', tempPwd)
    await typeTestId(page, 'settings-new-password', password)
    await clickTestId(page, 'settings-change-password')
    await waitForTestId(page, 'page-unlock', 10_000)
    await typeTestId(page, 'unlock-password', password)
    await clickTestId(page, 'unlock-submit')
    await waitForTestId(page, 'page-workspace')
    await collectCoverage(page)
  })
})
