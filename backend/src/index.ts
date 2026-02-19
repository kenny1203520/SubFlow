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
import { verifySession } from './middleware/auth';

dotenv.config();

import { SchedulerService } from './scheduler';
SchedulerService.init();

const app = express();
const httpServer = createServer(app);
const io = new Server(httpServer, {
    parser,
    cors: {
        origin: ["http://localhost:5173", "http://localhost:3000", process.env.FRONTEND_URL || ""],
        credentials: true
    }
});

const port = process.env.BACKEND_PORT || 3000;
const frontendPath = path.resolve(__dirname, '../../frontend/dist');

// Middleware
app.use(express.json());
app.use(cors({
    origin: ["http://localhost:5173", "http://localhost:3000", process.env.FRONTEND_URL || ""],
    credentials: true
}));
app.use(verifySession);

// Serve static files
app.use(express.static(frontendPath));

// API Routes
app.use("/auth", authRoutes);
app.use("/api/user", userRoutes);
app.use("/api/files", uploadRoutes);
app.use("/api/export", exportRoutes);
app.use("/api/audit", auditRoutes);

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
import { BillController } from './controllers/BillController';
import { WalletController } from './controllers/WalletController';
import { SecurityController } from './controllers/SecurityController';
import { ExpenseController } from './controllers/ExpenseController';
import { SubscriptionController } from './controllers/SubscriptionController';
import { NotificationController } from './controllers/NotificationController';
import { FileController } from './controllers/FileController';
import { ServiceController } from './controllers/ServiceController';

io.on("connection", (socket) => {
    // console.log(`User connected: ${socket.data.user.username}`);

    new GroupController(io, socket).register();
    new BillController(io, socket).register();
    new WalletController(io, socket).register();
    new SecurityController(io, socket).register();
    new ExpenseController(io, socket).register();
    new SubscriptionController(io, socket).register();
    new NotificationController(io, socket).register();
    new FileController(io, socket).register();
    new ServiceController(io, socket).register();

    socket.on("dashboard:stats", async (cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Total Owed TO you (people owe you money)
            const owedToMeRes = await pool.query(
                `SELECT SUM(es.amount_owed) as total
                 FROM expense_splits es
                 JOIN expenses e ON es.expense_id = e.id
                 WHERE e.paid_by = $1 AND es.user_id != $1 AND es.is_paid = false`,
                [userId]
            );

            // Total YOU owe (you owe people money)
            const iOweRes = await pool.query(
                `SELECT SUM(es.amount_owed) as total
                 FROM expense_splits es
                 WHERE es.user_id = $1 AND es.is_paid = false`,
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

    socket.on("ping", (cb) => {
        cb("pong");
    });

    socket.on("disconnect", () => {
        // console.log(`User disconnected: ${socket.data.user.username}`);
    });
});

// Fallback for SPA (Regex matches everything except /auth)
app.get(/^(?!\/auth).*/, (req, res, next) => {
    if (req.path.includes('.')) {
        return next();
    }
    res.sendFile(path.join(frontendPath, 'index.html'));
});

httpServer.listen(port, () => {
    // console.log(`BFF Server running at http://localhost:${port}`);
    // console.log(`Serving frontend from: ${frontendPath}`);
});