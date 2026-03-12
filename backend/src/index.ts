import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import { pool } from './db';
import parser from 'socket.io-msgpack-parser';

import authRoutes from './routes/auth';
import userRoutes from './routes/user';
import uploadRoutes from './routes/upload';
import exportRoutes from './routes/export';
import auditRoutes from './routes/audit';
import adminRoutes from './routes/admin';
import helmet from 'helmet';
import { verifySession, createSocketAuthenticationMiddleware, createSocketAuthorizationMiddleware } from './middleware/auth';
import { apiLimiter, createSocketRateLimitMiddleware } from './middleware/rateLimit';
import { SocketEventResponse, SOCKET_ERROR_CODES } from './types/socket-protocol';
import { systemSocketEvents } from './socket/events';

dotenv.config();

import { SchedulerService } from './scheduler';
SchedulerService.init();

const app = express();
app.set('trust proxy', 1);
const httpServer = createServer(app);
const io = new Server(httpServer, {
    parser,
    cors: {
        origin: ["http://localhost:5173", "http://localhost:3000", process.env.FRONTEND_URL || ""],
        credentials: true
    }
});

const port = process.env.BACKEND_PORT || 3000;

// Middleware
app.use(express.json());
app.use(helmet());
app.use(cors({
    origin: ["http://localhost:5173", "http://localhost:3000", process.env.FRONTEND_URL || ""],
    credentials: true
}));
app.use(apiLimiter);
app.use(verifySession);

// API Routes
app.use("/auth", authRoutes);
app.use("/api/user", userRoutes);
app.use("/api/files", uploadRoutes);
app.use("/api/export", exportRoutes);
app.use("/api/audit", auditRoutes);
app.use("/api/admin", adminRoutes);

// Static for uploads
app.use("/uploads", express.static(path.resolve(__dirname, '../../uploads')));

io.use(createSocketAuthenticationMiddleware());
io.use(createSocketRateLimitMiddleware());
io.use(createSocketAuthorizationMiddleware());

import { GroupController } from './controllers/GroupController';
import { GroupRoleController } from './controllers/GroupRoleController';
import { GroupPermissionController } from './controllers/GroupPermissionController';
import { BillController } from './controllers/BillController';
import { WalletController } from './controllers/WalletController';
import { SecurityController } from './controllers/SecurityController';
import { ExpenseController } from './controllers/ExpenseController';
import { SubscriptionController } from './controllers/SubscriptionController';
import { NotificationController } from './controllers/NotificationController';
import { FileController } from './controllers/FileController';
import { ServiceController } from './controllers/ServiceController';
import { AuthController } from './controllers/AuthController';

io.on("connection", (socket) => {
    // console.log(`User connected: ${socket.data.user.username}`);

    // ========================================================================================
    // 輔助函數：解析 Socket.IO 事件參數中的 callback (Helper to resolve ack callback)
    // ========================================================================================
    const resolveAck = (...args: any[]) => {
        const maybeAck = args[args.length - 1];
        return typeof maybeAck === 'function' ? maybeAck : null;
    };

    // ========================================================================================
    // 控制器初始化 (Controller Initialization)
    // ========================================================================================
    new AuthController(io, socket).register();
    new GroupController(io, socket).register();
    new GroupRoleController(io, socket).register();
    new GroupPermissionController(io, socket).register();
    new BillController(io, socket).register();
    new WalletController(io, socket).register();
    new SecurityController(io, socket).register();
    new ExpenseController(io, socket).register();
    new SubscriptionController(io, socket).register();
    new NotificationController(io, socket).register();
    new FileController(io, socket).register();
    new ServiceController(io, socket).register();

    // ========================================================================================
    // 系統事件：儀表板統計 (System Event: Dashboard Stats)
    // ========================================================================================
    socket.on(systemSocketEvents.DASHBOARD_STATS, async (...args: any[]) => {
        const cb = resolveAck(...args);
        if (!cb) return;

        try {
            const userId = socket.data.user.id;

            // 查詢用戶應收數額 (Query: total owed to user)
            const owedToMeRes = await pool.query(
                `SELECT 
                    (SELECT COALESCE(SUM(es.amount_owed), 0)
                     FROM expense_splits es
                     JOIN expenses e ON es.expense_id = e.id
                     JOIN group_members gm ON es.member_id = gm.id
                     WHERE e.paid_by = $1 
                       AND (gm.user_id != $1 OR gm.user_id IS NULL) 
                       AND es.status = 'pending'
                    ) +
                    (SELECT COALESCE(SUM(bs.amount_owed - bs.paid_amount), 0)
                     FROM bill_splits bs
                     JOIN bills b ON bs.bill_id = b.id
                     WHERE b.created_by = $1 AND bs.status = 'pending'
                    ) as total`,
                [userId]
            );

            // 查詢用戶應付數額 (Query: total owed by user)
            const iOweRes = await pool.query(
                `SELECT 
                    (SELECT COALESCE(SUM(es.amount_owed), 0)
                     FROM expense_splits es
                     JOIN expenses e ON es.expense_id = e.id
                     JOIN group_members gm ON es.member_id = gm.id
                     WHERE gm.user_id = $1 AND e.paid_by != $1 AND es.status = 'pending'
                    ) +
                    (SELECT COALESCE(SUM(bs.amount_owed - bs.paid_amount), 0)
                     FROM bill_splits bs
                     JOIN group_members gm ON bs.member_id = gm.id
                     WHERE gm.user_id = $1 AND bs.status = 'pending'
                    ) as total`,
                [userId]
            );

            // 查詢活躍訂閱數 (Query: active subscriptions count)
            const subCountRes = await pool.query(
                `SELECT COUNT(*) as count FROM subscriptions
                 WHERE owner_id = $1 AND status = 'active'`,
                [userId]
            );

            // 返回成功響應 (Return success response)
            cb(
                SocketEventResponse.success({
                    stats: {
                        totalOwedToMe: parseFloat(owedToMeRes.rows[0].total || 0),
                        totalIOwe: parseFloat(iOweRes.rows[0].total || 0),
                        activeSubscriptions: parseInt(subCountRes.rows[0].count || 0)
                    }
                })
            );
        } catch (error) {
            console.error("Error getting dashboard stats:", error);
            cb(
                SocketEventResponse.error(
                    SOCKET_ERROR_CODES.INTERNAL_ERROR,
                    "Failed to fetch stats"
                )
            );
        }
    });

    // ========================================================================================
    // 系統事件：心跳檢測 (System Event: Ping)
    // ========================================================================================
    socket.on(systemSocketEvents.PING, (...args: any[]) => {
        const cb = resolveAck(...args);
        if (cb) cb(SocketEventResponse.success({ pong: true }));
    });

    // ========================================================================================
    // 連線事件：用戶斷開連接 (Connection Event: User Disconnect)
    // ========================================================================================
    socket.on("disconnect", () => {
        // console.log(`User disconnected: ${socket.data.user.username}`);
    });
});

httpServer.listen(port, () => {
    // console.log(`[BFF Server] 運行在 http://localhost:${port}`);
    // console.log(`[Socket.IO] 安全通訊層已啟用`);
    // Serving frontend from: ${frontendPath}
});
