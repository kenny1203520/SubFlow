import { pool } from '../db';

export interface CaptchaSettings {
    enabled: boolean;
    provider: 'none' | 'turnstile' | 'recaptcha';
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

// In-memory cache to prevent constant DB queries for frequently accessed settings
const cache = new Map<string, { value: any; expiresAt: number }>();
const CACHE_TTL_MS = 1000 * 60 * 5; // 5 minutes

export class SystemSettingService {
    /**
     * Get a system setting by key, utilizing the cache.
     */
    static async getSetting<T>(key: string, defaultValue: T): Promise<T> {
        const cached = cache.get(key);
        if (cached && cached.expiresAt > Date.now()) {
            return cached.value as T;
        }

        try {
            const res = await pool.query('SELECT value FROM system_settings WHERE key = $1', [key]);
            
            if (res.rows.length > 0) {
                const value = res.rows[0].value as T;
                cache.set(key, { value, expiresAt: Date.now() + CACHE_TTL_MS });
                return value;
            }
            
            // If not found in DB, return default
            return defaultValue;
        } catch (error) {
            console.error(`Failed to get system setting ${key}:`, error);
            return defaultValue;
        }
    }

    /**
     * Update a system setting and invalidate the cache.
     */
    static async setSetting<T>(key: string, value: T, userId?: string): Promise<boolean> {
        try {
            await pool.query(
                `INSERT INTO system_settings (key, value, updated_by) 
                 VALUES ($1, $2, $3) 
                 ON CONFLICT (key) DO UPDATE 
                 SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = CURRENT_TIMESTAMP`,
                [key, JSON.stringify(value), userId || null]
            );
            
            // Update cache immediately
            cache.set(key, { value, expiresAt: Date.now() + CACHE_TTL_MS });
            return true;
        } catch (error) {
            console.error(`Failed to set system setting ${key}:`, error);
            return false;
        }
    }

    /**
     * Convenience methods for specific settings
     */
    static async getCaptchaSettings(): Promise<CaptchaSettings> {
        return this.getSetting<CaptchaSettings>('auth.captcha', {
            enabled: false,
            provider: 'none',
            siteKey: null,
            secretKey: null
        });
    }

    static async getPasswordPolicy(): Promise<PasswordPolicy> {
        return this.getSetting<PasswordPolicy>('auth.password_policy', {
            minLength: 8,
            requireUppercase: true,
            requireLowercase: true,
            requireNumbers: true,
            requireSymbols: true
        });
    }

    static async getTwoFactorSettings(): Promise<TwoFactorSettings> {
        return this.getSetting<TwoFactorSettings>('auth.require_2fa', {
            enabled: false
        });
    }
}
