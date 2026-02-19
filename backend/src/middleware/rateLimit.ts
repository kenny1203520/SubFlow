import rateLimit from 'express-rate-limit';

export const authLimiter = rateLimit({
    windowMs: Number(process.env.AUTH_RATE_LIMIT_WINDOW_MS) || 60 * 60 * 1000, // 60 minutes
    max: Number(process.env.AUTH_RATE_LIMIT_MAX) || 5, // Limit each IP to 5 requests per windowMs
    message: { error: "Too many login attempts, please try again after 60 minutes" },
    standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
    legacyHeaders: false, // Disable the `X-RateLimit-*` headers
});

export const apiLimiter = rateLimit({
    windowMs: Number(process.env.API_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000, // 15 minutes
    max: Number(process.env.API_RATE_LIMIT_MAX) || 100, // Limit each IP to 100 requests per windowMs
    message: { error: "Too many requests, please try again later" },
    standardHeaders: true,
    legacyHeaders: false,
});
