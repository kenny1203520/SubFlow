import { SystemSettingRepository } from '../repositories/SystemSettingRepository';

export interface CaptchaSettings {
    enabled: boolean;
    provider: 'none' | 'turnstile' | 'recaptcha';
    version?: string | null; // For reCAPTCHA, e.g., 'v2' or 'v3'
    siteKey: string | null;
    secretKey: string | null;
}

export interface PasswordPolicy {
    minLength: number;
    requireUppercase: boolean;
    requireLowercase: boolean;
    requireNumbers: boolean;
    requireSymbols: boolean;
}

export interface TwoFactorSettings {
    enabled: boolean;
}

export interface AuthLockoutSettings {
    maxFailedAttempts: number;
    lockoutDurationMins: number;
}

export interface RateLimitSettings {
    authWindowMs: number;
    authMax: number;
    apiWindowMs: number;
    apiMax: number;
}

export interface GroupSettings {
    groupsLimit: number;
    roleLimitPerGroup: number;
    memberLimitPerGroup: number;
}

// In-memory cache to prevent constant DB queries for frequently accessed settings
const cache = new Map<string, { value: any; expiresAt: number }>();
const CACHE_TTL_MS = 1000 * 60 * 5; // 5 minutes

export class SystemSettingService {
    private systemSettingRepo = new SystemSettingRepository();

    /**
     * Get a system setting by key, utilizing the cache.
     * @param key 
     * @param defaultValue 
     * @returns 
     */
    async getSetting<T>(key: string, defaultValue: T): Promise<T> {
        const cached = cache.get(key);
        if (cached && cached.expiresAt > Date.now()) {
            return cached.value as T;
        }

        try {
            const res = await this.systemSettingRepo.getSetting(key);
            
            // If not found in DB, return default
            if  (res === null) {
                return defaultValue;
            }
            const value = res.value as T;
            cache.set(key, { value, expiresAt: Date.now() + CACHE_TTL_MS });
            return value;
        } catch (error) {
            console.error(`Failed to get system setting ${key}:`, error);
            return defaultValue;
        }
    }

    /**
     * Update a system setting and invalidate the cache.
     * @param key 
     * @param value 
     * @param userId 
     * @param description The description is descriptive text about the setting, used for admin UI. It can be updated when changing the setting.
     * @returns 
     */
    async setSetting<T>(key: string, value: T, userId: string, description?: string): Promise<boolean> {
        try {
            await this.systemSettingRepo.setSetting(key, value, userId, description);
            
            // Update cache immediately
            cache.set(key, { value, expiresAt: Date.now() + CACHE_TTL_MS });
            return true;
        } catch (error) {
            console.error(`Failed to set system setting ${key}:`, error);
            return false;
        }
    }

    // Convenience methods for specific settings

    /**
     * Get CAPTCHA settings for authentication endpoints
     * @returns 
     */
    async getCaptchaSettings(): Promise<CaptchaSettings> {
        return this.getSetting<CaptchaSettings>('auth.captcha', {
            enabled: false,
            provider: 'none',
            version: null,
            siteKey: null,
            secretKey: null
        });
    }

    /**
     * Update CAPTCHA settings for authentication endpoints
     * @param settings 
     * @param userId 
     * @returns 
     */
    async updateCaptchaSettings(settings: CaptchaSettings, userId: string): Promise<boolean> {
        return this.setSetting('auth.captcha', settings, userId);
    }

    /**
     * Get global password policy settings
     * @returns 
     */
    async getPasswordPolicy(): Promise<PasswordPolicy> {
        return this.getSetting<PasswordPolicy>('auth.password_policy', {
            minLength: 8,
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: true
        });
    }

    /**
     * Update global password policy settings
     * @param policy 
     * @param userId 
     * @returns 
     */
    async updatePasswordPolicy(policy: PasswordPolicy, userId: string): Promise<boolean> {
        return this.setSetting('auth.password_policy', policy, userId);
    }

    /**
     * Get global 2FA requirement settings
     * @returns 
     */
    async getTwoFactorSettings(): Promise<TwoFactorSettings> {
        return this.getSetting<TwoFactorSettings>('auth.require_2fa', {
            enabled: false
        });
    }

    /**
     * Update global 2FA requirement settings
     * @param settings 
     * @param userId 
     * @returns 
     */
    async updateTwoFactorSettings(settings: TwoFactorSettings, userId: string): Promise<boolean> {
        return this.setSetting('auth.require_2fa', settings, userId);
    }

    async getAuthLockoutSettings(): Promise<AuthLockoutSettings> {
        return this.getSetting<AuthLockoutSettings>('security.auth_lockout', {
            maxFailedAttempts: 5,
            lockoutDurationMins: 720
        });
    }

    async updateAuthLockoutSettings(settings: AuthLockoutSettings, userId: string): Promise<boolean> {
        return this.setSetting('security.auth_lockout', settings, userId);
    }

    async getRateLimitSettings(): Promise<RateLimitSettings> {
        return this.getSetting<RateLimitSettings>('security.rate_limit', {
            authWindowMs: 15 * 60 * 1000, // 15 minutes
            authMax: 6,
            apiWindowMs: 5 * 60 * 1000, // 5 minutes
            apiMax: 1000
        });
    }

    async updateRateLimitSettings(settings: RateLimitSettings, userId: string): Promise<boolean> {
        return this.setSetting('security.rate_limit', settings, userId);
    }

    async getGroupSettings(): Promise<GroupSettings> {
        return this.getSetting<GroupSettings>('groups.settings', {
            groupsLimit: 100,
            roleLimitPerGroup: 10,
            memberLimitPerGroup: 1000
        });
    }

    async updateGroupSettings(settings: GroupSettings, userId: string): Promise<boolean> {
        return this.setSetting('groups.settings', settings, userId);
    }
}
