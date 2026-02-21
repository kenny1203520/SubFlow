import { GroupRepository, GroupRow } from '../repositories/GroupRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { GroupServiceRepository } from '../repositories/GroupServiceRepository';
import { UserRepository } from '../repositories/UserRepository';
import { GroupRoleRepository } from '../repositories/GroupRoleRepository';
import { NotificationService } from './NotificationService';
import { pool } from '../db';

export class GroupService {
    private groupRepo = new GroupRepository();
    private memberRepo = new GroupMemberRepository();
    private groupServiceRepo = new GroupServiceRepository();
    private userRepo = new UserRepository();
    private roleRepo = new GroupRoleRepository();
    private notifService = new NotificationService();

    async createGroup(userId: string, payload: any): Promise<GroupRow> {
        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            // 1. Create the Group Identity
            const group = await this.groupRepo.create({
                name: payload.name,
                description: payload.description,
                max_members: payload.max_members || 1,
                created_by: userId
            });

            // 2. Handle Service Creation if provided
            if (payload.service_name || payload.service_id) {
                // In a full implementation, we might want to check/create the global Service first
                // For now, let's just create the GroupService entry
                await this.groupServiceRepo.create({
                    group_id: group.id,
                    service_id: payload.service_id,
                    service_name: payload.service_name,
                    website: payload.website,
                    plan_name: payload.plan_name,
                    amount: payload.amount,
                    service_currency: payload.service_currency,
                    payment_currency: payload.payment_currency,
                    billing_type: payload.billing_type,
                    interval_unit: payload.interval_unit,
                    interval_value: payload.interval_value,
                    billing_method: payload.billing_method,
                    next_payment_date: (payload.next_payment_date && payload.next_payment_date !== '') ? new Date(payload.next_payment_date) : undefined,
                    created_by: userId
                });
            }

            // 3. Add the creator as Admin
            await this.memberRepo.addMember({
                group_id: group.id,
                user_id: userId,
                role: 'owner', // Changed from admin to owner to match schema roles better if needed
                joined_at: new Date(),
                created_by: userId
            });

            // 4. Handle initial members
            if (payload.initial_members) {
                for (const member of payload.initial_members) {
                    if (member.email || member.username) {
                        let user = null;
                        if (member.email) user = await this.userRepo.findByEmail(member.email);
                        else if (member.username) user = await this.userRepo.findByUsername(member.username);

                        if (user) {
                            // Invite via Notification
                            await this.memberRepo.addMember({
                                group_id: group.id,
                                user_id: user.id,
                                role: 'member',
                                created_by: userId
                            });

                            await this.notifService.createNotification(
                                user.id,
                                'group_invite',
                                'Group Invitation',
                                `You have been invited to join ${group.name}`,
                                { groupId: group.id, groupName: group.name, inviterId: userId }
                            );
                        } else {
                            // Fallback to temp member
                            await this.memberRepo.addMember({
                                group_id: group.id,
                                temp_name: member.email || member.username,
                                role: 'member',
                                created_by: userId
                            });
                        }
                    } else if (member.name) {
                        await this.memberRepo.addMember({
                            group_id: group.id,
                            temp_name: member.name,
                            role: 'member',
                            created_by: userId
                        });
                    }
                }
            }

            await client.query("COMMIT");
            return group;
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }

    async listGroups(userId: string): Promise<GroupRow[]> {
        return await this.groupRepo.findByUserId(userId);
    }

    async getGroupDetail(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("Not a member of this group");

        const group = await this.groupRepo.findById(groupId);
        let members = await this.memberRepo.getMembersByGroupId(groupId);
        const services = await this.groupServiceRepo.findByGroupId(groupId);

        // Fetch dynamic roles for members
        members = await Promise.all(members.map(async (m) => {
            const dynamicRoles = await this.roleRepo.getMemberRoles(m.member_id);
            return {
                ...m,
                dynamicRoles
            };
        }));

        return { group, members, services };
    }

    async addMember(adminId: string, groupId: string, payload: { email?: string, name?: string, username?: string }) {
        const role = await this.memberRepo.checkRole(groupId, adminId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can add members");

        if (payload.email || payload.username) {
            let user = null;
            if (payload.email) user = await this.userRepo.findByEmail(payload.email);
            else if (payload.username) user = await this.userRepo.findByUsername(payload.username);

            if (user) {
                const existingRole = await this.memberRepo.checkRole(groupId, user.id);
                if (existingRole) throw new Error("User is already a member of this group");

                const group = await this.groupRepo.findById(groupId);

                await this.memberRepo.addMember({
                    group_id: groupId,
                    user_id: user.id,
                    role: 'member',
                    created_by: adminId
                });

                await this.notifService.createNotification(
                    user.id,
                    'group_invite',
                    'Group Invitation',
                    `You have been invited to join ${group?.name}`,
                    { groupId: groupId, groupName: group?.name, inviterId: adminId }
                );
                return { status: 'invited' };
            } else {
                await this.memberRepo.addMember({
                    group_id: groupId,
                    temp_name: payload.email || payload.username,
                    role: 'member',
                    created_by: adminId
                });
                return { status: 'added_temp' };
            }
        } else if (payload.name) {
            await this.memberRepo.addMember({
                group_id: groupId,
                temp_name: payload.name,
                role: 'member',
                created_by: adminId
            });
            return { status: 'added_temp' };
        }
    }

    async bindMemberInvite(adminId: string, groupId: string, memberId: string, payload: { email?: string, username?: string }) {
        const role = await this.memberRepo.checkRole(groupId, adminId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can bind members");

        let user = null;
        if (payload.email) user = await this.userRepo.findByEmail(payload.email);
        else if (payload.username) user = await this.userRepo.findByUsername(payload.username);

        if (!user) throw new Error("User not found");

        const existingRole = await this.memberRepo.checkRole(groupId, user.id);
        if (existingRole) throw new Error("User is already a member of this group");

        const group = await this.groupRepo.findById(groupId);

        await this.notifService.createNotification(
            user.id,
            'group_bind_invite',
            'Group Slot Assignment',
            `You have been assigned to a slot in ${group?.name}. Accept to join?`,
            { groupId: groupId, memberId: memberId, groupName: group?.name, inviterId: adminId }
        );
    }


    async acceptInvite(userId: string, groupId: string, memberId?: string) {
        const members = await this.memberRepo.getMembersByGroupId(groupId);
        const existingMember = members.find(m => m.user_id === userId);

        if (existingMember && existingMember.joined_at) {
             throw new Error("Already a member");
        }

        if (memberId) {
            await this.memberRepo.bindMember(memberId, userId);
        } else if (existingMember) {
            await this.memberRepo.updateStatus(existingMember.member_id, new Date());
        } else {
             await this.memberRepo.addMember({
                group_id: groupId,
                user_id: userId,
                role: 'member',
                joined_at: new Date()
            });
        }
    }

    async cancelInvite(userId: string, groupId: string, memberId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can cancel invites");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found");
        if (member.joined_at) throw new Error("Member has already joined");

        await this.memberRepo.remove(memberId);
    }

    async bindMember(userId: string, memberId: string) {
        // Validation logic here
        await this.memberRepo.bindMember(memberId, userId);
    }

    async rejectInvite(userId: string, groupId: string) {
        // Just acknowledging the rejection. 
        // In the future, we might want to mark the notification as 'rejected/solved'
        return true;
    }

    async deleteGroup(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can delete the group");

        await this.groupRepo.delete(groupId);
    }

    async leaveGroup(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("You are not a member of this group");
        if (role === 'admin' || role === 'owner') throw new Error("Admins/Owners cannot leave the group. Delete it instead.");

        await this.memberRepo.removeMember(groupId, userId);
    }

    async updateGroup(userId: string, groupId: string, payload: any) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can update group settings");

        return await this.groupRepo.update(groupId, payload);
    }

    async updateMemberRole(userId: string, groupId: string, memberId: string, newRole: any) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can manage roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        // 1. Update legacy column for backward compatibility
        await this.memberRepo.updateRole(memberId, newRole);

        // 2. Update Dynamic Roles
        const adminRole = await this.roleRepo.findByName('Group Admin');
        if (adminRole) {
            if (newRole === 'admin') {
                await this.roleRepo.assignRoleToMember(memberId, adminRole.id, userId);
            } else {
                await this.roleRepo.removeRoleFromMember(memberId, adminRole.id);
            }
        }
    }

    async assignDynamicRole(userId: string, groupId: string, memberId: string, roleId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can manage dynamic roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        await this.roleRepo.assignRoleToMember(memberId, roleId, userId);
    }

    async removeDynamicRole(userId: string, groupId: string, memberId: string, roleId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can manage dynamic roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        await this.roleRepo.removeRoleFromMember(memberId, roleId);
    }

    async removeMember(userId: string, groupId: string, memberId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can remove members");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        if (member.user_id === userId) throw new Error("Cannot kick yourself. use Leave Group.");

        await this.memberRepo.remove(memberId);
    }

    async listRoles(): Promise<any[]> {
        return await this.roleRepo.listAll();
    }

    async createRole(userId: string, payload: { name: string; description?: string }): Promise<any> {
        return await this.roleRepo.create(payload.name, payload.description);
    }
}

