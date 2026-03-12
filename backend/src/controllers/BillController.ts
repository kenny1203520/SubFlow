import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { BillService } from '../services/BillService';
import { billSocketEvents } from '../socket/events';

export class BillController extends SocketController {
    private billService = new BillService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on(billSocketEvents.LIST, (payload, cb) => this.listBills(payload, cb));
        this.socket.on(billSocketEvents.GET, (payload, cb) => this.getBill(payload, cb));
        this.socket.on(billSocketEvents.UPDATE_SPLIT, (payload, cb) => this.updateSplit(payload, cb));
    }

    async listBills(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const bills = await this.billService.listBills(userId, payload.groupId);
            this.success(cb, { bills });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list bills");
        }
    }

    async getBill(payload: { billId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const detail = await this.billService.getBillDetail(userId, payload.billId);
            this.success(cb, detail);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get bill details");
        }
    }

    async updateSplit(payload: { splitId: string, amount: number }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.billService.updateSplit(userId, payload.splitId, payload.amount);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to update split");
        }
    }
}
