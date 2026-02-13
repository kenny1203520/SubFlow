import { createI18n } from 'vue-i18n';
import enCommon from './locales/en/common.json';
import enDashboard from './locales/en/dashboard.json';
import enAuth from './locales/en/auth.json';
import enGroups from './locales/en/groups.json';
import enSubscriptions from './locales/en/subscriptions.json';

import zhCommon from './locales/zh/common.json';
import zhDashboard from './locales/zh/dashboard.json';
import zhAuth from './locales/zh/auth.json';
import zhGroups from './locales/zh/groups.json';
import zhSubscriptions from './locales/zh/subscriptions.json';

// Define messages structure
const messages = {
    en: {
        common: enCommon,
        dashboard: enDashboard,
        auth: enAuth,
        groups: enGroups,
        subscriptions: enSubscriptions
    },
    zh: {
        common: zhCommon,
        dashboard: zhDashboard,
        auth: zhAuth,
        groups: zhGroups,
        subscriptions: zhSubscriptions
    }
};

const i18n = createI18n({
    legacy: false, // Composition API mode
    locale: 'zh', // Default locale
    fallbackLocale: 'en',
    messages
});

export default i18n;
