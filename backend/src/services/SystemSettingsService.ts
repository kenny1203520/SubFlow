import { pool } from '../db';

const CACHE_TTL_MS = 60_000; // 1-minute cache

interface CacheEntry {
    value: any;
    expiresAt: number;
}

const cache = new Map<string, CacheEntry>();

export const SystemSettingsService = {
    async getSetting(key: string): Promise<any> {
        const now = Date.now();
        const cached = cache.get(key);
        if (cached && cached.expiresAt > now) return cached.value;

        try {
            const res = await pool.query(
                'SELECT value FROM system_settings WHERE key = $1',
                [key]
            );
            const value = res.rows[0]?.value ?? null;
            cache.set(key, { value, expiresAt: now + CACHE_TTL_MS });
            return value;
        } catch {
            return null;
        }
    },

    async setSetting(key: string, value: any, userId?: string): Promise<void> {
        await pool.query(
            `INSERT INTO system_settings (key, value, updated_by)
             VALUES ($1, $2::jsonb, $3)
             ON CONFLICT (key) DO UPDATE SET value = $2::jsonb, updated_by = $3, updated_at = NOW()`,
            [key, JSON.stringify(value), userId ?? null]
        );
        cache.delete(key); // Invalidate cache on write
    },

    invalidate(key: string) {
        cache.delete(key);
    }
};
