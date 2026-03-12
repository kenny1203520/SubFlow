import { Server, Socket } from 'socket.io';

type AckFn = (res: any) => void;

export abstract class BaseController {
    protected io?: Server;
    protected socket?: Socket;

    constructor(io?: Server, socket?: Socket) {
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
