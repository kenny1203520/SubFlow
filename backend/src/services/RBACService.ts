import { RBACRepository } from '../repositories/RBACRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';

export class RBACService {
    private rbacRepo = new RBACRepository();
    private memberRepo = new GroupMemberRepository();

    async hasPermission(userId: string, groupId: string, permission: string): Promise<boolean> {
        const permissions = await this.rbacRepo.getMemberPermissions(groupId, userId);
        return permissions.includes(permission);
    }

    async checkPermission(userId: string, groupId: string, permission: string): Promise<void> {
        const allowed = await this.hasPermission(userId, groupId, permission);
        if (!allowed) throw new Error(`Missing permission: ${permission}`);
    }
}
