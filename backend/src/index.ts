import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import { pool } from './db';
import parser from 'socket.io-msgpack-parser';
import { lucia } from './auth/lucia';
import { parse } from 'cookie';

import authRoutes from './routes/auth';
import userRoutes from './routes/user';
import uploadRoutes from './routes/upload';
import exportRoutes from './routes/export';
import auditRoutes from './routes/audit';
import adminRoutes from './routes/admin';
import helmet from 'helmet';
import { verifySession } from './middleware/auth';
import { apiLimiter } from './middleware/rateLimit';

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

// Socket.IO Auth
io.use(async (socket, next) => {
    try {
        const cookieHeader = socket.handshake.headers.cookie;
        if (!cookieHeader) return next(new Error("Unauthorized"));

        const cookies = parse(cookieHeader);
        const sessionId = cookies[lucia.sessionCookieName];
        if (!sessionId) return next(new Error("Unauthorized"));

        const { session, user } = await lucia.validateSession(sessionId);
        if (!session) return next(new Error("Unauthorized"));

        socket.data.user = user;
        socket.data.session = session;
        next();
    } catch (e) {
        next(new Error("Authentication error"));
    }
});

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

    const resolveAck = (...args: any[]) => {
        const maybeAck = args[args.length - 1];
        return typeof maybeAck === 'function' ? maybeAck : null;
    };

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

    socket.on("dashboard:stats", async (...args: any[]) => {
        const cb = resolveAck(...args);
        if (!cb) return;

        try {
            const userId = socket.data.user.id;

            // Total Owed TO you (people owe you money)
            // Includes expenses you paid and bills you created
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

            // Total YOU owe (you owe people money)
            // Includes expense splits and bill splits where you are the member
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

            // Active subscriptions count
            const subCountRes = await pool.query(
                `SELECT COUNT(*) as count FROM subscriptions
                 WHERE owner_id = $1 AND status = 'active'`,
                [userId]
            );

            cb({
                status: "ok",
                stats: {
                    totalOwedToMe: parseFloat(owedToMeRes.rows[0].total || 0),
                    totalIOwe: parseFloat(iOweRes.rows[0].total || 0),
                    activeSubscriptions: parseInt(subCountRes.rows[0].count || 0)
                }
            });
        } catch (error) {
            console.error("Error getting dashboard stats:", error);
            cb({ status: "error", message: "Failed to fetch stats" });
        }
    });

    socket.on("ping", (...args: any[]) => {
        const cb = resolveAck(...args);
        if (cb) cb("pong");
    });

    socket.on("disconnect", () => {
        // console.log(`User disconnected: ${socket.data.user.username}`);
    });
});

httpServer.listen(port, () => {
    // console.log(`BFF Server running at http://localhost:${port}`);
    // console.log(`Serving frontend from: ${frontendPath}`);
});
