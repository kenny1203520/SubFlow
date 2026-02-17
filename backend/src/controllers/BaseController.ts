import { Server, Socket } from 'socket.io';

export abstract class BaseController {
    protected io: Server;
    protected socket: Socket;

    constructor(io: Server, socket: Socket) {
        this.io = io;
        this.socket = socket;
    }

    protected success(cb: (res: any) => void, data?: any) {
        cb({ status: 'ok', ...data });
    }

    protected error(cb: (res: any) => void, message: string) {
        cb({ status: 'error', message });
    }

    abstract register(): void;
}
