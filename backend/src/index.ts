import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import path from 'path';
import { createServer } from 'http';
import { Server } from 'socket.io';
import parser from 'socket.io-msgpack-parser';
import { lucia } from './auth/lucia';
import { parse } from 'cookie';

import authRoutes from './routes/auth';
import { verifySession } from './middleware/auth';

dotenv.config();

const app = express();
const httpServer = createServer(app);
const io = new Server(httpServer, {
    parser,
    cors: {
        origin: ["http://localhost:5173", "http://localhost:3000"],
        credentials: true
    }
});

const port = process.env.PORT || 3000;
const frontendPath = path.resolve(__dirname, '../../frontend/dist');

// Middleware
app.use(express.json());
app.use(cors({
    origin: ["http://localhost:5173", "http://localhost:3000"],
    credentials: true
}));

// Serve static files
app.use(express.static(frontendPath));

// API Routes
app.use("/auth", authRoutes);

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

import { registerGroupHandlers } from './socket/groups';
import { registerExpenseHandlers } from './socket/expenses';
import { registerSubscriptionHandlers } from './socket/subscriptions';

io.on("connection", (socket) => {
    console.log(`User connected: ${socket.data.user.username}`);

    registerGroupHandlers(io, socket);
    registerExpenseHandlers(io, socket);
    registerSubscriptionHandlers(io, socket);

    socket.on("ping", (cb) => {
        cb("pong");
    });

    socket.on("disconnect", () => {
        console.log(`User disconnected: ${socket.data.user.username}`);
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
    console.log(`BFF Server running at http://localhost:${port}`);
    console.log(`Serving frontend from: ${frontendPath}`);
});