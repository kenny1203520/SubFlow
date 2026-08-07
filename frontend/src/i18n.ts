import { computed, ref } from 'vue'
export type Locale = 'zh-TW' | 'en'
const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('subflow-locale') : null
export const locale = ref<Locale>(stored === 'en' ? 'en' : 'zh-TW')
const messages = { 'zh-TW': { profile:'個人設定', profileDesc:'管理你的個人資料與顯示偏好。', email:'Email', displayName:'顯示名稱', timezone:'時區', save:'儲存設定', saved:'已儲存', searchTimezone:'搜尋時區或城市…', noTimezone:'找不到符合的時區', currentTimezone:'目前時區', language:'語言', chooseLanguage:'選擇語言', chooseLanguageDesc:'選擇 SubFlow 介面使用的語言。', traditionalChinese:'繁體中文', traditionalChineseNative:'繁體中文', english:'English', englishNative:'英文', close:'關閉', searchLanguage:'搜尋語言…', noLanguage:'找不到符合的語言', theme:'外觀主題', themeDesc:'選擇介面的顯示模式。', systemTheme:'跟隨系統', lightTheme:'明亮模式', darkTheme:'深色模式', switchToLight:'切換為明亮模式', switchToDark:'切換為深色模式' }, en: { profile:'Profile', profileDesc:'Manage your personal information and display preferences.', email:'Email', displayName:'Display name', timezone:'Time zone', save:'Save settings', saved:'Saved', searchTimezone:'Search time zones or cities…', noTimezone:'No matching time zones', currentTimezone:'Current time zone', language:'Language', chooseLanguage:'Choose language', chooseLanguageDesc:'Choose the language used by the SubFlow interface.', traditionalChinese:'Traditional Chinese', traditionalChineseNative:'繁體中文', english:'English', englishNative:'English', close:'Close', searchLanguage:'Search languages…', noLanguage:'No matching languages', theme:'Appearance', themeDesc:'Choose how the interface is displayed.', systemTheme:'Follow system', lightTheme:'Light', darkTheme:'Dark', switchToLight:'Switch to light mode', switchToDark:'Switch to dark mode' } } as const
export function useI18n(){const t=computed(()=>messages[locale.value]);function setLocale(next:Locale){locale.value=next;localStorage.setItem('subflow-locale',next)}return{locale,t,setLocale}}




