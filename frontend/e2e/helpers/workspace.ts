import type { Page } from 'puppeteer'
import { clickTestId, typeTestId, waitForTestId, waitForText } from './browser.js'

export async function openWorkspace(page: Page): Promise<void> {
  await waitForTestId(page, 'page-workspace')
  await waitForTestId(page, 'workspace-flows')
}

export async function ensureFlow(page: Page): Promise<string> {
  await openWorkspace(page)
  if ((await page.$$('[data-testid^="flow-card-"]')).length === 0) {
    await clickTestId(page, 'flows-add')
  }
  await page.waitForSelector('[data-testid^="flow-card-"]')
  await page.click('[data-testid^="flow-card-"]')
  const id = await page.evaluate(() => {
    const card = document.querySelector('[data-testid^="flow-card-"]')
    return (card?.getAttribute('data-testid') ?? '').replace(/^flow-card-/, '')
  })
  if (!id) throw new Error('expected flow card id')
  return id
}

export async function firstOpId(page: Page): Promise<string> {
  await page.waitForSelector('[data-testid^="op-row-"]')
  const id = await page.evaluate(() => {
    const row = document.querySelector('[data-testid^="op-row-"]')
    return (row?.getAttribute('data-testid') ?? '').replace(/^op-row-/, '')
  })
  if (!id) throw new Error('expected operation edge')
  return id
}

export async function selectEdge(page: Page, opId: string): Promise<void> {
  const edge = await page.$(`[data-testid="canvas-edge-${opId}"]`)
  if (edge) await edge.click()
  await waitForTestId(page, `op-src-${opId}`)
}

export async function addLocalRemote(page: Page, name: string): Promise<void> {
  await page.waitForSelector('[data-testid^="canvas-node-"]')
  await page.click('[data-testid^="canvas-node-"]')
  await waitForTestId(page, 'workspace-remotes')
  await clickTestId(page, 'remotes-add')
  await waitForTestId(page, 'remotes-add-form')
  await typeTestId(page, 'remotes-name', name)
  await typeTestId(page, 'remotes-type', 'local')
  await clickTestId(page, 'remotes-submit')
  await waitForText(page, name, 15_000)
}

export async function listRemotes(page: Page): Promise<Array<{ name: string; type: string }>> {
  return page.evaluate(async () => {
    const r = await fetch('/api/v1/remotes', { credentials: 'same-origin' })
    if (!r.ok) throw new Error(`list remotes ${r.status}`)
    return (await r.json()) as Array<{ name: string; type: string }>
  })
}
