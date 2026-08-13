import { after, before, describe, it } from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
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
import { unlock } from '../helpers/auth.js'
import { ensureFlow, firstOpId, selectEdge } from '../helpers/workspace.js'

describe('flow run', () => {
  let page: Page
  let srcDir: string
  let dstDir: string

  before(async () => {
    srcDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gn-drive-e2e-src-'))
    dstDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gn-drive-e2e-dst-'))
    fs.writeFileSync(path.join(srcDir, 'hello.txt'), 'gn-drive e2e\n')
    page = await newPage()
    await unlock(page)
  })

  after(async () => {
    if (page) await closePage(page)
    await closeBrowser()
    for (const d of [srcDir, dstDir]) fs.rmSync(d, { recursive: true, force: true })
  })

  it('executes a push and copies the source file', async () => {
    const flowId = await ensureFlow(page)
    const opId = await firstOpId(page)
    await selectEdge(page, opId)
    await typeTestId(page, `op-src-${opId}`, srcDir)
    await typeTestId(page, `op-dst-${opId}`, dstDir)
    await clickTestId(page, `flows-save-bottom-${flowId}`)
    await new Promise((r) => setTimeout(r, 400))

    await Promise.all([
      page.waitForResponse((res) => res.url().includes('/execute') && res.ok(), { timeout: 15_000 }),
      clickTestId(page, 'flows-run'),
    ])

    const srcDisabled = await page.$eval(
      `[data-testid="op-src-${opId}"]`,
      (el) => (el as HTMLInputElement).disabled,
    )
    const bar = await page.$('[data-testid="flow-run-bar"]')
    if (bar) {
      assert.equal(srcDisabled, true, 'paths lock while the run bar is up')
    }

    await page.reload({ waitUntil: 'domcontentloaded' })
    await waitForTestId(page, 'page-workspace', 15_000)

    const marker = path.join(dstDir, 'hello.txt')
    const deadline = Date.now() + 45_000
    let copied = false
    while (Date.now() < deadline) {
      if (fs.existsSync(marker) && fs.readFileSync(marker, 'utf8').includes('gn-drive e2e')) {
        copied = true
        break
      }
      await new Promise((r) => setTimeout(r, 250))
    }
    assert.ok(copied, `expected ${marker} after execute`)

    await ensureFlow(page)
    await page.waitForFunction(
      () => !document.querySelector('[data-testid="flow-run-bar"]'),
      { timeout: 15_000 },
    )
    await collectCoverage(page)
  })
})
