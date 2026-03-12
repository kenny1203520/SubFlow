/**
 * Socket.IO 中間件層
 * 統一處理安全、授權、隱私和審計
 */

import { Server, Socket } from 'socket.io';
import { pool } from '../db';
import { SOCKET_AUTH_RULES, SOCKET_ERROR_CODES, SENSITIVE_FIELDS, SocketResponse, SocketEventResponse } from '../types/socket-protocol';
import { logActivity } from '../utils/audit';

// ============================================================================
// 速率限制
// ============================================================================

const rateLimitStore = new Map<string, { timestamp: number; count: number }[]>();

export function createRateLimitMiddleware(io: Server) {
    return async (socket: Socket, next: (err?: Error) => void) => {
        const userId = socket.data.user?.id;
        if (!userId) {
            return next(new Error('User not authenticated'));
        }

        // 監聽事件時檢查速率限制
        socket.onAny((eventName: string, ...args) => {
            const rule = SOCKET_AUTH_RULES[eventName];
            if (!rule?.rateLimit) return;

            const key = `${userId}:${eventName}`;
            const now = Date.now();
            const oneMinuteAgo = now - 60000;
            const oneHourAgo = now - 3600000;

            // 初始化或清理過期記錄
            if (!rateLimitStore.has(key)) {
                rateLimitStore.set(key, []);
            }

            const timestamps = rateLimitStore.get(key)!;
            const recentTimestamps = timestamps.filter(t => t.timestamp > oneHourAgo);

            // 檢查分鐘限制 (Check minute limit)
            const lastMinute = recentTimestamps.filter(t => t.timestamp > oneMinuteAgo).length;
            if (lastMinute >= rule.rateLimit.requestsPerMinute) {
                const cb = args[args.length - 1];
                if (typeof cb === 'function') {
                    cb(SocketEventResponse.rateLimitExceeded());
                }
                return;
            }

            // 檢查小時限制 (Check hour limit)
            if (recentTimestamps.length >= rule.rateLimit.requestsPerHour) {
                const cb = args[args.length - 1];
                if (typeof cb === 'function') {
                    cb(SocketEventResponse.rateLimitExceeded());
                }
                return;
            }

            // 記錄此次請求
            recentTimestamps.push({ timestamp: now, count: 1 });
            rateLimitStore.set(key, recentTimestamps);
        });

        next();
    };
}

// ============================================================================
// 授權檢查
// ============================================================================

export function createAuthorizationMiddleware(io: Server) {
    return async (socket: Socket, next: (err?: Error) => void) => {
        socket.onAny(async (eventName: string, ...args) => {
            const rule = SOCKET_AUTH_RULES[eventName];

            if (!rule) {
                // 未定義的事件也應該記錄
                const cb = args[args.length - 1];
                if (typeof cb === 'function') {
                    cb({
                        status: 'error',
                        error: {
                            code: SOCKET_ERROR_CODES.INTERNAL_ERROR,
                            message: 'Event not recognized'
                        }
                    } as SocketResponse);
                }
                return;
            }

            // 認證檢查
            if (rule.requiresAuthentication && !socket.data.user) {
                const cb = args[args.length - 1];
                if (typeof cb === 'function') {
                    cb({
                        status: 'error',
                        error: {
                            code: SOCKET_ERROR_CODES.UNAUTHORIZED,
                            message: 'Authentication required'
                        }
                    } as SocketResponse);
                }
                return;
            }

            // 權限檢查
            if (rule.requiredPermission && socket.data.user) {
                const hasPermission = await checkPermission(
                    socket.data.user.id,
                    rule.requiredPermission,
                    args[0]?.groupId  // 如果有 groupId，檢查群組級權限
                );

                if (!hasPermission) {
                    const cb = args[args.length - 1];
                    if (typeof cb === 'function') {
                        cb({
                            status: 'error',
                            error: {
                                code: SOCKET_ERROR_CODES.PERMISSION_DENIED,
                                message: 'Insufficient permissions for this operation'
                            }
                        } as SocketResponse);
                    }

                    // 記錄未授權的嘗試
                    await logActivity(
                        socket.data.user.id,
                        'SOCKET_AUTH_FAILED',
                        'UNAUTHORIZED_SOCKET_ACCESS',
                        'medium',
                        `Unauthorized Socket.IO access attempt for event: ${eventName}`,
                        socket.request,
                        undefined,
                        {
                            event: eventName,
                            timestamp: new Date()
                        }
                    );
                    return;
                }
            }
        });

        next();
    };
}

// ============================================================================
// 數據隱私過濾
// ============================================================================

export function sanitizeResponse(data: any, dataType?: string): any {
    if (!data) return data;

    if (Array.isArray(data)) {
        return data.map(item => sanitizeResponse(item, dataType));
    }

    if (typeof data !== 'object') {
        return data;
    }

    const sanitized = { ...data };

    // 移除敏感欄位
    const fieldsToRemove = dataType ? SENSITIVE_FIELDS[dataType as keyof typeof SENSITIVE_FIELDS] : [];
    if (fieldsToRemove) {
        fieldsToRemove.forEach(field => {
            delete sanitized[field];
        });
    }

    // 遞迴處理嵌套物件
    Object.keys(sanitized).forEach(key => {
        if (typeof sanitized[key] === 'object' && sanitized[key] !== null) {
            sanitized[key] = sanitizeResponse(sanitized[key], dataType);
        }
    });

    return sanitized;
}

// ============================================================================
// 驗證
// ============================================================================

export async function validateSocketPayload(eventName: string, payload: any): Promise<{ valid: boolean; error?: string }> {
    // 可以根據事件類型進行特定驗證
    // 這裡是簡單示例，實際應該根據 eventName 使用相應的 Zod schema

    if (!payload || typeof payload !== 'object') {
        return { valid: true }; // 某些事件不需要 payload
    }

    // 基本檢查：如果是需要 payload 的事件，檢查必要欄位
    if (eventName.includes('add') || eventName.includes('create') || eventName.includes('update')) {
        if (Object.keys(payload).length === 0 && !eventName.includes('list')) {
            return { valid: false, error: 'Payload cannot be empty for this operation' };
        }
    }

    return { valid: true };
}

// ============================================================================
// 審計日誌
// ============================================================================

export async function logSocketEvent(
    userId: string,
    eventName: string,
    action: 'success' | 'failure',
    details?: any
): Promise<void> {
    try {
        await pool.query(
            `INSERT INTO activity_logs (user_id, action, resource_type, status, metadata, created_at)
             VALUES ($1, $2, $3, $4, $5, NOW())`,
            [userId, `SOCKET_${eventName.toUpperCase()}`, 'SOCKET_EVENT', action, JSON.stringify(details || {})]
        );
    } catch (error) {
        console.error('Failed to log socket event:', error);
    }
}

// ============================================================================
// 輔助函數
// ============================================================================

async function checkPermission(userId: string, permission: string, groupId?: string): Promise<boolean> {
    try {
        // 獲取用戶的全局權限
        const globalCheckQuery = `
            SELECT 1 FROM user_permissions 
            WHERE user_id = $1 AND permission_name = $2
        `;
        const globalResult = await pool.query(globalCheckQuery, [userId, permission]);

        if (globalResult.rows.length > 0) {
            return true;
        }

        // 如果有 groupId，檢查群組級權限
        if (groupId) {
            const groupCheckQuery = `
                SELECT 1 FROM group_member_permissions 
                WHERE user_id = $1 AND group_id = $2 AND permission_name = $3
            `;
            const groupResult = await pool.query(groupCheckQuery, [userId, groupId, permission]);
            return groupResult.rows.length > 0;
        }

        return false;
    } catch (error) {
        console.error('Permission check failed:', error);
        return false;
    }
}

// ============================================================================
// 異常處理
// ============================================================================

export function createErrorHandlingMiddleware(io: Server) {
    return async (socket: Socket, next: (err?: Error) => void) => {
        socket.onAnyOutgoing((packet) => {
            // 在每個發出的事件上應用隱私過濾
            if (packet.data && packet.data[0]?.data) {
                // 根據需要應用 sanitizeResponse
            }
        });

        next();
    };
}
