/**
 * Socket.IO 傳輸協議定義（僅保留通訊契約）
 * 事件命名與授權規則由 socket/events 與 socket/policies 管理
 */

import { z } from 'zod';

// ============================================================================
// 請求格式
// ============================================================================

export const SocketRequestSchema = z.object({
    type: z.enum(['query', 'mutation', 'action']).default('action'),
    resource: z.string().min(1).max(50),
    action: z.string().min(1).max(50),
    payload: z.record(z.string(), z.any()).optional(),
    requestId: z.string().optional(),
    timestamp: z.number().optional()
});

export type SocketRequest = z.infer<typeof SocketRequestSchema>;

// ============================================================================
// 響應格式
// ============================================================================

export const SocketResponseSchema = z.object({
    status: z.enum(['ok', 'error']),
    requestId: z.string().optional(),
    data: z.record(z.string(), z.any()).optional(),
    error: z.object({
        code: z.string(),
        message: z.string(),
        details: z.record(z.string(), z.any()).optional()
    }).optional(),
    timestamp: z.number()
});

export type SocketResponse = z.infer<typeof SocketResponseSchema>;

// ============================================================================
// 錯誤碼統一定義
// ============================================================================

export const SOCKET_ERROR_CODES = {
    // 認證錯誤 (1000-1099)
    UNAUTHORIZED: 'AUTH_001',
    SESSION_EXPIRED: 'AUTH_002',
    INVALID_CREDENTIALS: 'AUTH_003',
    
    // 授權錯誤 (1100-1199)
    PERMISSION_DENIED: 'AUTHZ_001',
    INSUFFICIENT_PRIVILEGES: 'AUTHZ_002',
    RESOURCE_ACCESS_DENIED: 'AUTHZ_003',
    
    // 驗證錯誤 (1200-1299)
    VALIDATION_FAILED: 'VALIDATION_001',
    INVALID_PAYLOAD: 'VALIDATION_002',
    MISSING_REQUIRED_FIELD: 'VALIDATION_003',
    
    // 資源錯誤 (1300-1399)
    NOT_FOUND: 'RESOURCE_001',
    ALREADY_EXISTS: 'RESOURCE_002',
    CONFLICT: 'RESOURCE_003',
    
    // 業務邏輯錯誤 (1400-1499)
    BUSINESS_LOGIC_ERROR: 'BUSINESS_001',
    INVALID_STATE_TRANSITION: 'BUSINESS_002',
    OPERATION_NOT_ALLOWED: 'BUSINESS_003',
    
    // 系統錯誤 (1500-1599)
    INTERNAL_ERROR: 'SYSTEM_001',
    SERVICE_UNAVAILABLE: 'SYSTEM_002',
    RATE_LIMIT_EXCEEDED: 'SYSTEM_003'
} as const;

// ============================================================================
// 敏感欄位定義（需隱藏或加密）
// ============================================================================

export const SENSITIVE_FIELDS = {
    user: ['password_hash', 'email_verified_at', 'failed_login_attempts', 'last_login_ip'],
    security: ['two_factor_secret', 'backup_codes', 'totp_token'],
    payment: ['credit_card_encrypted', 'bank_account_encrypted', 'stripe_customer_id'],
    audit: ['ip_address', 'user_agent', 'device_fingerprint']
} as const;

// ============================================================================
// Socket.IO 事件響應工具類 (Socket.IO Event Response Helper Class)
// ============================================================================

export class SocketEventResponse {
    /**
     * 成功響應產生器 (Success response generator)
     */
    static success(data?: any, requestId?: string): SocketResponse {
        return {
            status: 'ok',
            data,
            requestId,
            timestamp: Date.now()
        };
    }

    /**
     * 錯誤響應產生器 (Error response generator)
     */
    static error(
        code: string,
        message: string,
        details?: any,
        requestId?: string
    ): SocketResponse {
        return {
            status: 'error',
            error: {
                code,
                message,
                details
            },
            requestId,
            timestamp: Date.now()
        };
    }

    /**
     * 未授權錯誤 (Unauthorized error)
     */
    static unauthorized(requestId?: string): SocketResponse {
        return this.error(
            SOCKET_ERROR_CODES.UNAUTHORIZED,
            'Unauthorized access',
            undefined,
            requestId
        );
    }

    /**
     * 禁止訪問錯誤 (Forbidden error)
     */
    static forbidden(requestId?: string): SocketResponse {
        return this.error(
            SOCKET_ERROR_CODES.PERMISSION_DENIED,
            'Access forbidden',
            undefined,
            requestId
        );
    }

    /**
     * 驗證失敗錯誤 (Validation error)
     */
    static validationError(message: string, details?: any, requestId?: string): SocketResponse {
        return this.error(
            SOCKET_ERROR_CODES.VALIDATION_FAILED,
            message,
            details,
            requestId
        );
    }

    /**
     * 未找到資源錯誤 (Not found error)
     */
    static notFound(resource: string, requestId?: string): SocketResponse {
        return this.error(
            SOCKET_ERROR_CODES.NOT_FOUND,
            `${resource} not found`,
            undefined,
            requestId
        );
    }

    /**
     * 速率限制超超 (Rate limit exceeded)
     */
    static rateLimitExceeded(requestId?: string): SocketResponse {
        return this.error(
            SOCKET_ERROR_CODES.RATE_LIMIT_EXCEEDED,
            'Too many requests',
            undefined,
            requestId
        );
    }
}
