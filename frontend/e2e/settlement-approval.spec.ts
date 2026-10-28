import { expect, test } from '@playwright/test'
import { randomUUID } from 'node:crypto'
import { account } from './fixtures/session'

test('settlement confirmation stays pending until a manager gives a reason and confirms it', async ({ page, request }, testInfo) => {
  test.setTimeout(120_000)
  const { headers, userID } = await account(page, request)
  const suffix = randomUUID()
  const createdGroup = await request.post('/api/subflow/v1/groups', { headers, data: {
    name: `Approval E2E ${suffix}`, currency: 'TWD', timezone: 'Asia/Taipei',
  } })
  expect(createdGroup.status()).toBe(201)
  const group = (await createdGroup.json()).data

  const updatedGroup = await request.patch(`/api/subflow/v1/groups/${group.id}`, { headers, data: {
    name: group.name, description: group.description, currency: group.currency, timezone: group.timezone,
    color: group.color, attachmentMaxSize: group.attachmentMaxSize || 0, attachmentMaxCount: group.attachmentMaxCount || 0,
    settlementApprovalEnabled: true,
  } })
  expect(updatedGroup.status()).toBe(200)

  const createdPlaceholder = await request.post(`/api/subflow/v1/groups/${group.id}/temp-members`, { headers, data: { name: `Pending payer ${suffix}` } })
  expect(createdPlaceholder.status()).toBe(201)
  const placeholderID = (await createdPlaceholder.json()).data.userId as string
  const expense = await request.post('/api/subflow/v1/expenses', { headers, data: {
    groupId: group.id, title: `Shared expense ${suffix}`, amountMinor: 10000, currency: 'TWD',
    paidBy: placeholderID, incurredOn: new Date().toISOString(), splitMode: 'equal',
    splits: [{ userId: userID }, { userId: placeholderID }],
  } })
  expect(expense.status()).toBe(201)

  const createdSettlement = await request.post(`/api/subflow/v1/groups/${group.id}/settlements`, { headers, data: {
    fromUserId: userID, toUserId: placeholderID, amountMinor: 5000, settledOn: new Date().toISOString(),
  } })
  expect(createdSettlement.status()).toBe(201)
  const settlement = (await createdSettlement.json()).data
  expect(settlement.status).toBe('pending')
  expect(settlement.approvalRequired).toBe(true)

  const reviewsResponse = await request.get(`/api/subflow/v1/groups/${group.id}/settlement-reviews`, { headers })
  expect(reviewsResponse.status()).toBe(200)
  const review = (await reviewsResponse.json()).data.find((item: { settlementId: string }) => item.settlementId === settlement.id)
  expect(review).toBeTruthy()

  const missingReason = await request.post(`/api/subflow/v1/settlement-reviews/${review.id}/respond`, { headers, data: { decision: 'accept', reason: '' } })
  expect(missingReason.status()).toBe(400)
  const stillPending = await request.get(`/api/subflow/v1/groups/${group.id}/settlements?perPage=25`, { headers })
  expect((await stillPending.json()).data.find((item: { id: string }) => item.id === settlement.id).status).toBe('pending')

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/home')
  const action = page.locator('.home-action-item').filter({ hasText: 'Settlement awaiting confirmation' })
  await expect(action).toBeVisible()
  await action.getByRole('button', { name: 'Confirm', exact: true }).click()
  const reasonForm = action.locator('form.home-reject-form')
  await expect(reasonForm.getByLabel('Reason for confirming on behalf of the recipient')).toBeVisible()
  const reason = 'Manager recorded and verified this payment.'
  await reasonForm.getByLabel('Reason for confirming on behalf of the recipient').fill(reason)
  const responsePromise = page.waitForResponse(response => response.url().includes(`/settlement-reviews/${review.id}/respond`) && response.request().method() === 'POST')
  await reasonForm.getByRole('button', { name: 'Confirm', exact: true }).click()
  const decisionResponse = await responsePromise
  expect(decisionResponse.status(), await decisionResponse.text()).toBe(200)
  const decision = (await decisionResponse.json()).data
  expect(decision.status).toBe('accepted')
  expect(decision.decidedBy).toBe(userID)
  expect(decision.reason).toBe(reason)

  const settlements = await request.get(`/api/subflow/v1/groups/${group.id}/settlements?perPage=25`, { headers })
  expect(settlements.status()).toBe(200)
  expect((await settlements.json()).data.find((item: { id: string }) => item.id === settlement.id).status).toBe('confirmed')
  await page.screenshot({ path: testInfo.outputPath('settlement-confirmed-home-mobile.png'), fullPage: true, animations: 'disabled' })
})
