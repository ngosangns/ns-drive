import { after, before, describe, it } from 'node:test'
import assert from 'node:assert/strict'
import type { Page } from 'puppeteer'
import {
  clickTestId,
  closeBrowser,
  closePage,
  collectCoverage,
  confirmDialog,
  goto,
  newPage,
  typeTestId,
  waitForTestId,
} from '../helpers/browser.js'
import { lockFromSettings, lockFromTopbar, unlock } from '../helpers/auth.js'

describe('auth', () => {
  let page: Page

  before(async () => {
    page = await newPage()
  })

  after(async () => {
    if (page) await closePage(page)
    await closeBrowser()
  })

  it('unlocks into the canvas workspace', async () => {
    await unlock(page)
    await waitForTestId(page, 'page-workspace')
    await waitForTestId(page, 'workspace-flows')
    await waitForTestId(page, 'flow-canvas')
    await collectCoverage(page)
  })

  it('rejects a wrong password and a short password', async () => {
    await lockFromSettings(page)
    await typeTestId(page, 'unlock-password', 'definitely-wrong-password')
    await clickTestId(page, 'unlock-submit')
    await page.waitForFunction(
      () => {
        const err = document.querySelector('[data-testid="unlock-error"]')
        return !!err && (err.textContent?.length ?? 0) > 0
      },
      { timeout: 10_000 },
    )
    await typeTestId(page, 'unlock-password', 'ab')
    await clickTestId(page, 'unlock-submit')
    await new Promise((r) => setTimeout(r, 300))
    assert.ok(await page.$('[data-testid="page-unlock"]'))
    await unlock(page)
    await waitForTestId(page, 'page-workspace')
    await collectCoverage(page)
  })

  it('locks from the topbar and guards unknown routes', async () => {
    await clickTestId(page, 'lock-button')
    await confirmDialog(page, 'Lock')
    await waitForTestId(page, 'page-unlock')
    await goto(page, '/profiles')
    await waitForTestId(page, 'page-unlock')
    await collectCoverage(page)
  })
})
