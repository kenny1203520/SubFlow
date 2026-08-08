import type { Category } from './api/types'
import type { MessageKey } from './i18n'

const systemKeys: Record<string, MessageKey> = {
  food_dining: 'category_food_dining', transport: 'category_transport', housing: 'category_housing', utilities: 'category_utilities',
  shopping: 'category_shopping', entertainment: 'category_entertainment', health: 'category_health', education: 'category_education',
  travel: 'category_travel', insurance: 'category_insurance', software_digital: 'category_software_digital', memberships: 'category_memberships',
  taxes_fees: 'category_taxes_fees', gifts_donations: 'category_gifts_donations', other: 'category_other',
}

export function categoryLabel(category: Pick<Category, 'systemKey' | 'customName'> | undefined, fallback: string, tr: (key: MessageKey) => string) {
  const key = category?.systemKey ? systemKeys[category.systemKey] : undefined
  return key ? tr(key) : category?.customName || fallback || tr('uncategorized')
}
export function categoryIcon(category: Pick<Category, 'systemKey' | 'iconKey'> | undefined) { return category?.iconKey || category?.systemKey || 'tag' }
