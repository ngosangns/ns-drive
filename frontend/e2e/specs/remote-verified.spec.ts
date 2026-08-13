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
  waitForText,
} from '../helpers/browser.js'
import { unlock } from '../helpers/auth.js'
import { addLocalRemote, ensureFlow, listRemotes } from '../helpers/workspace.js'

describe('remote verified add', () => {
  let page: Page
  const localName = `e2e_local_${Date.now().toString(36)}`

  before(async () => {
    page = await newPage()
    await unlock(page)
    await ensureFlow(page)
  })

  after(async () => {
    if (page) await closePage(page)
    await closeBrowser()
  })

  it('adds a local remote after a successful probe', async () => {
    await addLocalRemote(page, localName)
    const remotes = await listRemotes(page)
    assert.ok(
      remotes.some((r) => r.name === localName && r.type === 'local'),
      `GET /remotes should include ${localName}: ${JSON.stringify(remotes)}`,
    )
    await collectCoverage(page)
  })

  it('does not create a mega remote without credentials', async () => {
    await page.click('[data-testid^="canvas-node-"]')
    await waitForTestId(page, 'workspace-remotes')
    if (!(await page.$('[data-testid="remotes-add-form"]'))) {
      await clickTestId(page, 'remotes-add')
    }
    await waitForTestId(page, 'remotes-add-form')
    const megaName = `e2e_mega_${Date.now().toString(36)}`
    await typeTestId(page, 'remotes-name', megaName)
    await typeTestId(page, 'remotes-type', 'mega')
    await waitForTestId(page, 'remotes-cfg-user')
    await waitForTestId(page, 'remotes-cfg-pass')
    await clickTestId(page, 'remotes-submit')
    await new Promise((r) => setTimeout(r, 400))
    const remotes = await listRemotes(page)
    assert.ok(
      !remotes.some((r) => r.name === megaName),
      `mega without creds must not be stored: ${JSON.stringify(remotes)}`,
    )
    assert.ok(await page.$('[data-testid="remotes-add-form"]'))
    await collectCoverage(page)
    void waitForText
  })
})
