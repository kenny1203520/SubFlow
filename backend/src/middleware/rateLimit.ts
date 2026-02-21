import rateLimit from 'express-rate-limit';
import { SystemSettingsService } from '../services/SystemSettingsService';

// Dynamic rate limit that reads from DB (cached). Fallback to env vars for cold-start safety.
async function getRateLimitConfig() {
    const cfg = await SystemSettingsService.getSetting('security.rate_limit');
    return {
        authMax: cfg?.authMax ?? Number(process.env.AUTH_RATE_LIMIT_MAX)    ?? 5,
        apiMax:  cfg?.apiMax  ?? Number(process.env.API_RATE_LIMIT_MAX)     ?? 100,
    };
}

export const authLimiter = rateLimit({
    windowMs: Number(process.env.AUTH_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000,
    max: async (_req, _res) => {
        const cfg = await getRateLimitConfig();
        return cfg.authMax;
    }, // Limit each IP to X requests per windowMs
    message: { message: 'auth.errors.tooManyAttempts' },
    standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
    legacyHeaders: false, // Disable the `X-RateLimit-*` headers
});

export const apiLimiter = rateLimit({
    windowMs: Number(process.env.API_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000,
    max: async (_req, _res) => {
        const cfg = await getRateLimitConfig();
        return cfg.apiMax;
    }, // Limit each IP to X requests per windowMs
    message: { message: 'auth.errors.tooManyRequests' },
    standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
    legacyHeaders: false, // Disable the `X-RateLimit-*` headers
});
