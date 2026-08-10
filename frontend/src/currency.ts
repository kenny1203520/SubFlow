import type { Currency } from './api/types'

const defaultLocale=()=>typeof document==='undefined'?'zh-TW':document.documentElement.lang||'zh-TW'
export function currencyName(code:Currency,locale=defaultLocale()){
  try{return new Intl.NumberFormat(locale,{style:'currency',currency:code,currencyDisplay:'name'}).formatToParts(0).find(part=>part.type==='currency')?.value||code}catch{return code}
}
export function currencyLabel(code:Currency,locale=defaultLocale()){return `${currencyName(code,locale)} (${code})`}
