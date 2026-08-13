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
import { ensureFlow, firstOpId, openWorkspace, selectEdge } from '../helpers/workspace.js'

describe('canvas edit', () => {
  let page: Page
  let srcDir: string

  before(async () => {
    srcDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gn-drive-e2e-canvas-'))
    page = await newPage()
    await unlock(page)
  })

  after(async () => {
    if (page) await closePage(page)
    await closeBrowser()
    fs.rmSync(srcDir, { recursive: true, force: true })
  })

  it('creates a flow with two location nodes and one sync edge', async () => {
    await openWorkspace(page)
    const flowId = await ensureFlow(page)
    await page.waitForSelector('[data-testid^="canvas-node-"]')
    const nodes = await page.$$('[data-testid^="canvas-node-"]')
    assert.ok(nodes.length >= 2, `expected ≥2 nodes, got ${nodes.length}`)
    const opId = await firstOpId(page)
    assert.ok(await page.$(`[data-testid="canvas-edge-${opId}"]`))
    await collectCoverage(page)
    void flowId
  })

  it('persists an edited source path across reload', async () => {
    const flowId = await ensureFlow(page)
    const opId = await firstOpId(page)
    await selectEdge(page, opId)
    await typeTestId(page, `op-src-${opId}`, srcDir)
    await clickTestId(page, `flows-save-bottom-${flowId}`)
    await page.waitForResponse(
      (res) => res.url().includes('/flows/') && res.request().method() === 'PUT' && res.ok(),
      { timeout: 10_000 },
    ).catch(() => undefined)
    await new Promise((r) => setTimeout(r, 400))

    await page.reload({ waitUntil: 'domcontentloaded' })
    await waitForTestId(page, 'page-workspace', 15_000)
    await ensureFlow(page)
    await selectEdge(page, opId)
    const srcVal = await page.$eval(
      `[data-testid="op-src-${opId}"]`,
      (el) => (el as HTMLInputElement).value,
    )
    assert.ok(srcVal.includes(srcDir), `source should persist, got ${srcVal}`)
    await collectCoverage(page)
  })
})
