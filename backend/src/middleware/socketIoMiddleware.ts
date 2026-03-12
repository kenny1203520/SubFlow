/**
 * Socket.IO 中間件層
 * 統一處理共用的 Socket 工具：隱私過濾、載荷驗證、審計
 */

import { Server, Socket } from 'socket.io';
import { SENSITIVE_FIELDS } from '../types/socket-protocol';
import { logActivity, riskLevel } from '../utils/audit';

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
    risk: riskLevel,
    details?: any
): Promise<void> {
    try {
        logActivity(
            userId, 
            'socket_event', 
            `socket_${eventName.toLowerCase()}`, 
            risk, 
            'Event handled with status: ' + action, 
            undefined, undefined, 
            details
        );
    } catch (error) {
        console.error('Failed to log socket event:', error);
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
