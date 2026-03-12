import { Server, Socket } from 'socket.io';

type AckFn = (res: any) => void;

/**
 * SocketController is a base class for Socket.IO event handlers that REQUIRE socket connectivity.
 * Use this for controllers that are ONLY used in Socket.IO mode (not HTTP).
 * 
 * Unlike BaseController where socket/io is optional, SocketController enforces that both are required.
 * This prevents silent failures if socket is accidentally undefined.
 */
export abstract class SocketController {
    protected io: Server;
    protected socket: Socket;

    constructor(io: Server, socket: Socket) {
        if (!io) throw new Error("SocketController requires io (Server) to be provided");
        if (!socket) throw new Error("SocketController requires socket (Socket) to be provided");
        
        this.io = io;
        this.socket = socket;
    }

    protected resolveAck(...args: any[]): AckFn | null {
        const maybeAck = args[args.length - 1];
        return typeof maybeAck === 'function' ? maybeAck : null;
    }

    protected success(cb: unknown, data?: any) {
        if (typeof cb !== 'function') return;
        cb({ status: 'ok', ...data });
    }

    protected error(cb: unknown, message: string) {
        if (typeof cb !== 'function') return;
        cb({ status: 'error', message });
    }

    abstract register(): void;
}
