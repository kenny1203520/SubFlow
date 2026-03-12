/**
 * 標準化 Socket.IO 事件處理基類
 * 所有 Socket.IO 事件必須透過此基類，確保一致的安全和隱私標準
 */

import { Server, Socket } from 'socket.io';
import { randomUUID } from 'crypto';
import {
    SocketRequest,
    SocketResponse,
    SOCKET_ERROR_CODES,
    SocketEventResponse
} from '../types/socket-protocol';
import { socketPolicyRegistry } from '../socket/registry/SocketPolicyRegistry';
import {
    sanitizeResponse,
    validateSocketPayload,
    logSocketEvent
} from '../middleware/socketIoMiddleware';

/**
 * Socket 事件處理的響應回調簽名 (Ack callback signature)
 */
type AckCallback = (response: SocketResponse) => void;

/**
 * Socket 事件處理器函數簽名 (Event handler function signature)
 */
type SocketEventHandler = (
    payload: any,
    ack: AckCallback | null
) => Promise<void> | void;

/**
 * 標準化 Socket.IO 事件註冊基類
 */
export class StandardSocketEventHandler {
    protected io: Server;
    protected socket: Socket;
    protected userId: string;

    constructor(io: Server, socket: Socket) {
        this.io = io;
        this.socket = socket;
        this.userId = socket.data.user?.id || '';
    }

    /**
     * 安全地註冊事件監聽
     * 自動處理：認證、授權、驗證、隱私、審計
     */
    protected registerEvent(
        eventName: string,
        handler: SocketEventHandler
    ): void {
        this.socket.on(eventName, async (...args: any[]) => {
            const requestId = randomUUID();
            const payload = this.extractPayload(args);
            const ack = this.extractAck(args);

            try {
                // 1. 檢查認證規則
                const rule = socketPolicyRegistry.getAuthRule(eventName);
                if (!rule) {
                    const response = SocketEventResponse.error(
                        SOCKET_ERROR_CODES.INTERNAL_ERROR,
                        'Event not recognized',
                        undefined,
                        requestId
                    );
                    this.sendAck(ack, response);
                    return;
                }

                // 2. 驗證認證狀態
                if (rule.requiresAuthentication && !this.userId) {
                    const response = SocketEventResponse.unauthorized(requestId);
                    this.sendAck(ack, response);
                    await logSocketEvent(
                        'SYSTEM',
                        eventName,
                        'failure',
                        'medium',
                        { reason: 'Unauthorized', requestId }
                    );
                    return;
                }

                // 3. 驗證載荷
                const validation = await validateSocketPayload(eventName, payload);
                if (!validation.valid) {
                    const response = SocketEventResponse.validationError(
                        validation.error || 'Invalid payload',
                        undefined,
                        requestId
                    );
                    this.sendAck(ack, response);
                    return;
                }

                // 4. 執行業務邏輯
                await handler(payload, (response: SocketResponse) => {
                    // 應用隱私過濾
                    const sanitized = sanitizeResponse(response.data);
                    const filteredResponse = {
                        ...response,
                        data: sanitized,
                        requestId
                    };

                    this.sendAck(ack, filteredResponse);

                    // 5. 記錄審計
                    if (this.userId) {
                        logSocketEvent(
                            this.userId,
                            eventName,
                            response.status === 'ok' ? 'success' : 'failure',
                            'low',
                            {
                                requestId,
                                error: response.error
                            }
                        ).catch(console.error);
                    }
                });
            } catch (error: any) {
                console.error(`Socket event error [${eventName}]:`, error);

                const response = SocketEventResponse.error(
                    SOCKET_ERROR_CODES.INTERNAL_ERROR,
                    error.message || 'Internal server error',
                    undefined,
                    requestId
                );

                this.sendAck(ack, response);

                if (this.userId) {
                    await logSocketEvent(
                        this.userId,
                        eventName,
                        'failure',
                        'high',
                        {
                            requestId,
                            error: error.message
                        }
                    );
                }
            }
        });
    }

    /**
     * 從可變參數中提取有效載荷
     */
    private extractPayload(args: any[]): any {
        if (args.length === 0) return undefined;

        // 如果最後一個是函式（callback），則 payload 是前面的
        const lastArg = args[args.length - 1];
        if (typeof lastArg === 'function') {
            return args[args.length - 2];
        }

        // 否則最後一個可能是 payload
        return args[0];
    }

    /**
     * 從可變參數中提取 callback
     */
    private extractAck(args: any[]): AckCallback | null {
        const lastArg = args[args.length - 1];
        return typeof lastArg === 'function' ? lastArg : null;
    }

    /**
     * 安全地發送確認回應
     */
    private sendAck(ack: AckCallback | null, response: SocketResponse): void {
        if (ack && typeof ack === 'function') {
            try {
                ack(response);
            } catch (error) {
                console.error('Error sending ack:', error);
            }
        }
    }

    /**
     * 實作了標準響應格式的通用成功回應
     */
    protected reply(ack: AckCallback | null, data?: any, requestId?: string): void {
        const response = SocketEventResponse.success(data, requestId);
        this.sendAck(ack, response);
    }

    /**
     * 實作了標準響應格式的通用錯誤回應
     */
    protected replyError(
        ack: AckCallback | null,
        code: string,
        message: string,
        details?: any,
        requestId?: string
    ): void {
        const response = SocketEventResponse.error(code, message, details, requestId);
        this.sendAck(ack, response);
    }

    /**
     * 廣播訊息到群組中的所有使用者（帶隱私過濾）
     */
    protected broadcastToGroup(
        groupId: string,
        event: string,
        data: any,
        excludeUserId?: string
    ): void {
        const sanitized = sanitizeResponse(data);

        this.io.to(`group:${groupId}`).emit(event, {
            status: 'broadcast',
            data: sanitized,
            timestamp: Date.now()
        });
    }

    /**
     * 向單個使用者發送私人訊息
     */
    protected sendPrivate(userId: string, event: string, data: any): void {
        const sanitized = sanitizeResponse(data);

        // 發送給特定使用者的 socket
        this.io.to(`user:${userId}`).emit(event, {
            status: 'private_message',
            data: sanitized,
            timestamp: Date.now(),
            recipient: userId
        });
    }
}

/**
 * 使用示例：
 *
 * export class GroupController extends StandardSocketEventHandler {
 *     register() {
 *         this.registerEvent('group:list', (payload, ack) => this.listGroups(ack));
 *         this.registerEvent('group:create', (payload, ack) => this.createGroup(payload, ack));
 *     }
 *
 *     private async listGroups(ack: AckCallback | null) {
 *         try {
 *             const groups = await this.fetchGroups();
 *             this.reply(ack, { groups });
 *         } catch (error: any) {
 *             this.replyError(ack, SOCKET_ERROR_CODES.INTERNAL_ERROR, error.message);
 *         }
 *     }
 *
 *     private async createGroup(payload: any, ack: AckCallback | null) {
 *         const validation = validateGroupPayload(payload);
 *         if (!validation.valid) {
 *             this.replyError(
 *                 ack,
 *                 SOCKET_ERROR_CODES.VALIDATION_FAILED,
 *                 validation.error
 *             );
 *             return;
 *         }
 *
 *         const group = await this.createGroupInDb(payload);
 *         this.reply(ack, { group });
 *         this.broadcastToGroup(group.id, 'group:updated', { group });
 *     }
 * }
 */
