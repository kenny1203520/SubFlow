import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { ServiceService } from '../services/ServiceService';
import { serviceSocketEvents } from '../socket/events';

export class ServiceController extends SocketController {
    private serviceService = new ServiceService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on(serviceSocketEvents.SEARCH, (payload, cb) => this.searchServices(payload, cb));
        this.socket.on(serviceSocketEvents.CREATE, (payload, cb) => this.createService(payload, cb));
        this.socket.on(serviceSocketEvents.LIST, (...args: any[]) => this.listServices(this.resolveAck(...args) as any));
        this.socket.on(serviceSocketEvents.UPDATE, (payload, cb) => this.updateService(payload, cb));
        this.socket.on(serviceSocketEvents.DELETE, (payload, cb) => this.deleteService(payload, cb));
    }

    async searchServices(payload: { query: string }, cb: (res: any) => void) {
        try {
            const services = await this.serviceService.searchServices(payload.query);
            this.success(cb, { services });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to search services");
        }
    }

    async createService(payload: { name: string, domain?: string, icon_url?: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const service = await this.serviceService.createService(userId, payload);
            this.success(cb, { service });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to create service");
        }
    }

    async listServices(cb: (res: any) => void) {
        try {
            const services = await this.serviceService.listServices();
            this.success(cb, { services });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list services");
        }
    }

    async updateService(payload: { id: string, name?: string, domain?: string, icon_url?: string }, cb: (res: any) => void) {
        try {
            const service = await this.serviceService.updateService(payload.id, payload);
            this.success(cb, { service });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to update service");
        }
    }

    async deleteService(payload: { id: string }, cb: (res: any) => void) {
        try {
            await this.serviceService.deleteService(payload.id);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to delete service");
        }
    }
}
