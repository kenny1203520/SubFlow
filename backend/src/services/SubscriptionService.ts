import { SubscriptionRepository } from '../repositories/SubscriptionRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';

export class SubscriptionService {
    private subRepo = new SubscriptionRepository();
    // private memberRepo = new GroupMemberRepository(); // Removed as subscriptions are now personal

    async addSubscription(userId: string, payload: { name: string, amount: number, cycle: 'monthly' | 'yearly', startDate: string }) {
        // Direct personal subscription creation
        return await this.subRepo.create({
            owner_id: userId,
            service_name: payload.name,
            amount: payload.amount,
            cycle: payload.cycle,
            next_payment_date: payload.startDate ? new Date(payload.startDate) : undefined
        });
    }

    async listSubscriptions(userId: string, groupId: string) {
        // Deprecated functionality for now as per schema
        // If we want to show group subscriptions, we need a different approach or schema update.
        // For now, return empty or redirect to personal.
        return [];
    }

    async listAllSubscriptions(userId: string) {
        return await this.subRepo.findAllByUserId(userId);
    }

    async updateStatus(userId: string, subscriptionId: string, status: string) {
        // TODO: Add ownership check (is this user the owner of the subscription?)
        // For now, proceeding with update.
        const allowedStatus = ["active", "paused", "cancelled"];
        if (!allowedStatus.includes(status)) {
            throw new Error("Invalid status");
        }

        await this.subRepo.updateStatus(subscriptionId, status);
    }
}
