import rateLimit from 'express-rate-limit';

const authWindowMs = Number(process.env.AUTH_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000;
export const authLimiter = rateLimit({
    windowMs: authWindowMs, // 15 minutes default
    max: Number(process.env.AUTH_RATE_LIMIT_MAX) || 5, // Limit each IP to 5 requests per windowMs
    message: { message: "auth.errors.tooManyAttempts" },
    standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
    legacyHeaders: false, // Disable the `X-RateLimit-*` headers
});

const apiWindowMs = Number(process.env.API_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000;
export const apiLimiter = rateLimit({
    windowMs: apiWindowMs, // 15 minutes default
    max: Number(process.env.API_RATE_LIMIT_MAX) || 100, // Limit each IP to 100 requests per windowMs
    message: { message: "auth.errors.tooManyRequests" },
    standardHeaders: true,
    legacyHeaders: false,
});
