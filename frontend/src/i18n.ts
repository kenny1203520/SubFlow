import { createI18n } from 'vue-i18n';
import enCommon from './locales/en/common.json';
import enDashboard from './locales/en/dashboard.json';
import enAuth from './locales/en/auth.json';
import enGroups from './locales/en/groups.json';
import enSubscriptions from './locales/en/subscriptions.json';
import enProfile from './locales/en/profile.json';
import enSecurity from './locales/en/security.json';
import enAdmin from './locales/en/admin.json';
import enWallet from './locales/en/wallet.json';
import enActivity from './locales/en/activity.json';

import zhCommon from './locales/zh/common.json';
import zhDashboard from './locales/zh/dashboard.json';
import zhAuth from './locales/zh/auth.json';
import zhGroups from './locales/zh/groups.json';
import zhSubscriptions from './locales/zh/subscriptions.json';
import zhProfile from './locales/zh/profile.json';
import zhSecurity from './locales/zh/security.json';
import zhAdmin from './locales/zh/admin.json';
import zhWallet from './locales/zh/wallet.json';
import zhActivity from './locales/zh/activity.json';

// Define messages structure
const messages = {
    en: {
        common: enCommon,
        dashboard: enDashboard,
        auth: enAuth,
        groups: enGroups,
        subscriptions: enSubscriptions,
        profile: enProfile,
        security: enSecurity,
        admin: enAdmin,
        wallet: enWallet,
        activity: enActivity
    },
    zh: {
        common: zhCommon,
        dashboard: zhDashboard,
        auth: zhAuth,
        groups: zhGroups,
        subscriptions: zhSubscriptions,
        profile: zhProfile,
        security: zhSecurity,
        admin: zhAdmin,
        wallet: zhWallet,
        activity: zhActivity
    }
};

const i18n = createI18n({
    legacy: false, // Composition API mode
    locale: 'zh', // Default locale
    fallbackLocale: 'en',
    messages
});

export default i18n;
