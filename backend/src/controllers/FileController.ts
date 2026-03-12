import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { FileService } from '../services/FileService';

export class FileController extends SocketController {
    private fileService = new FileService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on("file:get", (payload, cb) => this.getFile(payload, cb));
        this.socket.on("file:delete", (payload, cb) => this.deleteFile(payload, cb));
    }

    async getFile(payload: { fileId: string }, cb: (res: any) => void) {
        try {
            const file = await this.fileService.getFile(payload.fileId);
            if (!file) return this.error(cb, "File not found");
            this.success(cb, { file });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get file info");
        }
    }

    async deleteFile(payload: { fileId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.fileService.deleteFile(payload.fileId, userId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to delete file");
        }
    }
}
