import rateLimit from 'express-rate-limit';
import { Socket } from 'socket.io';
import { SystemSettingService } from '../services/SystemSettingService';
import { SocketEventResponse } from '../types/socket-protocol';
import { socketPolicyRegistry } from '../socket/registry/SocketPolicyRegistry';

type HttpRateLimitConfig = {
    authMax: number;
    apiMax: number;
};

export type RateLimitRule = {
    requestsPerMinute: number;
    requestsPerHour: number;
};

export type SocketRateLimitConfig = {
    default?: Partial<RateLimitRule>;
    events?: Record<string, Partial<RateLimitRule>>;
};

const systemSettingService = new SystemSettingService();
const rateLimitStore = new Map<string, number[]>();
const SOCKET_RATE_LIMIT_SETTING_KEY = 'security.socket_rate_limit';
const SOCKET_RATE_LIMIT_CACHE_TTL_MS = 60_000;
const DEFAULT_SOCKET_RATE_LIMIT: RateLimitRule = {
    requestsPerMinute: 60,
    requestsPerHour: 1000
};

let socketRateLimitConfigCache: { value: SocketRateLimitConfig; expiresAt: number } | null = null;

async function getRateLimitConfig(): Promise<HttpRateLimitConfig> {
    const cfg = await systemSettingService.getRateLimitSettings();
    return {
        authMax: cfg.authMax ?? Number(process.env.AUTH_RATE_LIMIT_MAX) ?? 5,
        apiMax: cfg.apiMax ?? Number(process.env.API_RATE_LIMIT_MAX) ?? 100,
    };
}

function extractSocketAck(args: any[]): ((response: any) => void) | null {
    const maybeAck = args[args.length - 1];
    return typeof maybeAck === 'function' ? maybeAck : null;
}

function normalizeRateLimitRule(input: Partial<RateLimitRule> | undefined, fallback: RateLimitRule): RateLimitRule {
    const requestsPerMinute = Number(input?.requestsPerMinute);
    const requestsPerHour = Number(input?.requestsPerHour);

    return {
        requestsPerMinute: Number.isFinite(requestsPerMinute) && requestsPerMinute > 0
            ? Math.floor(requestsPerMinute)
            : fallback.requestsPerMinute,
        requestsPerHour: Number.isFinite(requestsPerHour) && requestsPerHour > 0
            ? Math.floor(requestsPerHour)
            : fallback.requestsPerHour
    };
}

async function refreshSocketRateLimitConfig(force = false): Promise<SocketRateLimitConfig> {
    const now = Date.now();
    if (!force && socketRateLimitConfigCache && socketRateLimitConfigCache.expiresAt > now) {
        return socketRateLimitConfigCache.value;
    }

    const fallback: SocketRateLimitConfig = {
        default: DEFAULT_SOCKET_RATE_LIMIT,
        events: {}
    };
    const config = await systemSettingService.getSetting<SocketRateLimitConfig>(SOCKET_RATE_LIMIT_SETTING_KEY, fallback);
    const normalized: SocketRateLimitConfig = {
        default: normalizeRateLimitRule(config?.default, DEFAULT_SOCKET_RATE_LIMIT),
        events: config?.events ?? {}
    };

    socketRateLimitConfigCache = {
        value: normalized,
        expiresAt: now + SOCKET_RATE_LIMIT_CACHE_TTL_MS
    };

    return normalized;
}

function resolveSocketRateLimit(eventName: string): RateLimitRule {
    const config = socketRateLimitConfigCache?.value;
    const defaultRule = normalizeRateLimitRule(config?.default, DEFAULT_SOCKET_RATE_LIMIT);
    return normalizeRateLimitRule(config?.events?.[eventName], defaultRule);
}

export const authLimiter = rateLimit({
    windowMs: Number(process.env.AUTH_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000,
    max: async (_req, _res) => {
        const cfg = await getRateLimitConfig();
        return cfg.authMax;
    },
    message: { message: 'auth.errors.tooManyAttempts' },
    standardHeaders: true,
    legacyHeaders: false,
});

export const apiLimiter = rateLimit({
    windowMs: Number(process.env.API_RATE_LIMIT_WINDOW_MS) || 15 * 60 * 1000,
    max: async (_req, _res) => {
        const cfg = await getRateLimitConfig();
        return cfg.apiMax;
    },
    message: { message: 'auth.errors.tooManyRequests' },
    standardHeaders: true,
    legacyHeaders: false,
});

export function createSocketRateLimitMiddleware() {
    return async (socket: Socket, next: (err?: Error) => void) => {
        await refreshSocketRateLimitConfig();

        socket.onAny((eventName: string, ...args) => {
            if (!socketPolicyRegistry.hasAuthRule(eventName)) {
                return;
            }

            if (!socketRateLimitConfigCache || socketRateLimitConfigCache.expiresAt <= Date.now()) {
                void refreshSocketRateLimitConfig();
            }

            const ack = extractSocketAck(args);
            const rule = resolveSocketRateLimit(eventName);
            const identityKey = socket.data.user?.id ?? socket.id;
            const storeKey = `${identityKey}:${eventName}`;
            const now = Date.now();
            const oneMinuteAgo = now - 60_000;
            const oneHourAgo = now - 3_600_000;
            const timestamps = rateLimitStore.get(storeKey) ?? [];
            const recentTimestamps = timestamps.filter((timestamp) => timestamp > oneHourAgo);
            const lastMinuteCount = recentTimestamps.filter((timestamp) => timestamp > oneMinuteAgo).length;

            if (lastMinuteCount >= rule.requestsPerMinute || recentTimestamps.length >= rule.requestsPerHour) {
                ack?.(SocketEventResponse.rateLimitExceeded());
                return;
            }

            recentTimestamps.push(now);
            rateLimitStore.set(storeKey, recentTimestamps);
        });

        next();
    };
}