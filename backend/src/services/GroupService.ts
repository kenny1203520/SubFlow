import { GroupRepository, GroupRow } from '../repositories/GroupRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { GroupServiceRepository } from '../repositories/GroupServiceRepository';
import { UserRepository } from '../repositories/UserRepository';
import { GroupRoleRepository } from '../repositories/GroupRoleRepository';
import { NotificationService } from './NotificationService';
import { SystemSettingsService } from './SystemSettingsService';
import { pool } from '../db';
import { PoolClient } from 'pg';

export class GroupService {
    private groupRepo = new GroupRepository();
    private memberRepo = new GroupMemberRepository();
    private groupServiceRepo = new GroupServiceRepository();
    private userRepo = new UserRepository();
    private roleRepo = new GroupRoleRepository();
    private notifService = new NotificationService();

    private async createDefaultGroupRoles(client: PoolClient, groupId: string): Promise<any[]> {
        const defaultRoles = [
            { name: 'Group Owner', description: 'Full control over the group, including billing and deletion.' },
            { name: 'Group Admin', description: 'Can manage group settings and members.' },
            { name: 'Group Treasurer', description: 'Can manage expenses and group balances.' },
            { name: 'Group Member', description: 'Basic member with read-only access and expense creation rights.' },
            { name: 'Group Viewer', description: 'Read-only access to group information.' }
        ];

        const createdRoles: any[] = [];
        for (const role of defaultRoles) {
            try {
                const result = await client.query(
                    `INSERT INTO group_roles (group_id, name, description, is_system_role)
                     VALUES ($1, $2, $3, true)
                     ON CONFLICT (group_id, name)
                     DO UPDATE SET description = EXCLUDED.description, is_system_role = true
                     RETURNING *`,
                    [groupId, role.name, role.description]
                );
                if (result.rows[0]) {
                    createdRoles.push(result.rows[0]);
                }
            } catch (err: any) {
                throw err;
            }
        }

        // Group Owner: all group permissions
        await client.query(
            `INSERT INTO permissions_group_role (role_id, permission_id)
             SELECT gr.id, p.id
             FROM group_roles gr
             CROSS JOIN permissions p
             WHERE gr.group_id = $1 AND gr.name = 'Group Owner' AND p.scope = 'group'
             ON CONFLICT DO NOTHING`,
            [groupId]
        );

        // Group Admin: all except ownership transfer and group deletion
        await client.query(
            `INSERT INTO permissions_group_role (role_id, permission_id)
             SELECT gr.id, p.id
             FROM group_roles gr
             CROSS JOIN permissions p
             WHERE gr.group_id = $1 AND gr.name = 'Group Admin'
             AND p.scope = 'group'
             AND (
               (p.action IN ('read', 'create', 'update', 'delete', 'invite', 'remove', 'assign', 'grant', 'revoke', 'manage', 'settle', 'upload', 'export'))
               AND p.resource NOT IN ('group', 'ownership')
             )
             ON CONFLICT DO NOTHING`,
            [groupId]
        );

        // Group Treasurer: finance/expenses + read basics
        await client.query(
            `INSERT INTO permissions_group_role (role_id, permission_id)
             SELECT gr.id, p.id
             FROM group_roles gr
             CROSS JOIN permissions p
             WHERE gr.group_id = $1 AND gr.name = 'Group Treasurer'
             AND p.scope = 'group'
             AND (
               p.resource IN ('expenses', 'finance', 'subscriptions', 'services', 'billing') OR
               (p.action = 'read' AND p.resource IN ('details', 'members', 'files'))
             )
             ON CONFLICT DO NOTHING`,
            [groupId]
        );

        // Group Member: read basics + create expenses
        await client.query(
            `INSERT INTO permissions_group_role (role_id, permission_id)
             SELECT gr.id, p.id
             FROM group_roles gr
             CROSS JOIN permissions p
             WHERE gr.group_id = $1 AND gr.name = 'Group Member'
             AND p.scope = 'group'
             AND (
               (p.action = 'read' AND p.resource IN ('details', 'members', 'services', 'subscriptions', 'expenses', 'finance', 'files')) OR
               (p.action = 'create' AND p.resource = 'expenses')
             )
             ON CONFLICT DO NOTHING`,
            [groupId]
        );

        // Group Viewer: read-only
        await client.query(
            `INSERT INTO permissions_group_role (role_id, permission_id)
             SELECT gr.id, p.id
             FROM group_roles gr
             CROSS JOIN permissions p
             WHERE gr.group_id = $1 AND gr.name = 'Group Viewer'
             AND p.scope = 'group' AND p.action = 'read'
             ON CONFLICT DO NOTHING`,
            [groupId]
        );

        return createdRoles;
    }

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

            // 2.5 Create default group roles
            const createdRoles = await this.createDefaultGroupRoles(client, group.id);

            // 3. Add the creator as Owner
            await this.memberRepo.addMember({
                group_id: group.id,
                user_id: userId,
                role: 'owner',
                joined_at: new Date(),
                created_by: userId
            });

            // 3.1 Assign Group Owner role to the creator
            const memberOwner = await this.memberRepo.findByGroupAndUser(group.id, userId);
            const ownerRole = createdRoles.find(r => r.name === 'Group Owner');
            if (ownerRole && ownerRole.id && memberOwner && memberOwner.id) {
                try {
                    await this.roleRepo.assignRoleToMember(memberOwner.id, ownerRole.id, userId, client);
                } catch (err: any) {
                    throw err;
                }
            }

            // 4. Handle initial members
            const memberRole = createdRoles.find(r => r.name === 'Group Member');
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

                            const memberX = await this.memberRepo.findByGroupAndUser(group.id, user.id);
                            if (memberRole && memberRole.id && memberX && memberX.id) {
                                try {
                                    await this.roleRepo.assignRoleToMember(memberX.id, memberRole.id, userId, client);
                                } catch (err: any) {
                                    throw new Error(err?.message);
                                }
                            }

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
        } catch (error: any) {
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
        const adminRole = await this.roleRepo.findByName(groupId, 'Group Admin');
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

        const targetRole = await this.roleRepo.findById(roleId);
        if (!targetRole || targetRole.group_id !== groupId) throw new Error("Role not found in this group");

        await this.roleRepo.assignRoleToMember(memberId, roleId, userId);
    }

    async removeDynamicRole(userId: string, groupId: string, memberId: string, roleId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can manage dynamic roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        const targetRole = await this.roleRepo.findById(roleId);
        if (!targetRole || targetRole.group_id !== groupId) throw new Error("Role not found in this group");

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

    async listRoles(groupId: string): Promise<any[]> {
        return await this.roleRepo.listAll(groupId);
    }

    async createRole(userId: string, groupId: string, payload: { name: string; description?: string }): Promise<any> {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin' && role !== 'owner') throw new Error("Only admins can create roles");

        const cfg = await SystemSettingsService.getSetting('groups.role_limit');
        const maxRoles = typeof cfg === 'number' ? cfg : (cfg?.max ?? null);
        if (typeof maxRoles === 'number' && maxRoles > 0) {
            const currentCount = await this.roleRepo.countByGroupId(groupId);
            if (currentCount >= maxRoles) {
                throw new Error("Role limit reached for this group");
            }
        }

        return await this.roleRepo.create(groupId, payload.name, payload.description);
    }
    

    /**
     * Transfer group ownership from current owner to another member
     * Only the current owner can transfer ownership
     */
    async transferOwnership(currentOwnerId: string, groupId: string, newOwnerId: string): Promise<void> {
        // 1. Verify the current user is the owner
        const currentOwnerMember = await this.memberRepo.findByGroupAndUser(groupId, currentOwnerId);
        if (!currentOwnerMember || currentOwnerMember.role !== 'owner') {
            throw new Error('Only the group owner can transfer ownership');
        }

        // 2. Verify the new owner is a member of the group
        const newOwnerMember = await this.memberRepo.findByGroupAndUser(groupId, newOwnerId);
        if (!newOwnerMember) {
            throw new Error('Target user is not a member of this group');
        }

        if (!newOwnerMember.joined_at) {
            throw new Error('Target user has not accepted the group invitation yet');
        }

        // 3. Use transaction to ensure atomic update
        const client = await pool.connect();
        try {
            await client.query('BEGIN');

            // Demote current owner to admin
            await client.query(
                `UPDATE group_members SET role = 'admin', updated_at = CURRENT_TIMESTAMP 
                 WHERE id = $1`,
                [currentOwnerMember.id]
            );

            // Promote new owner
            await client.query(
                `UPDATE group_members SET role = 'owner', updated_at = CURRENT_TIMESTAMP 
                 WHERE id = $1`,
                [newOwnerMember.id]
            );

            // Update group's created_by to reflect new owner
            await client.query(
                `UPDATE groups SET created_by = $2, updated_at = CURRENT_TIMESTAMP 
                 WHERE id = $1`,
                [groupId, newOwnerId]
            );

            await client.query('COMMIT');

            // Send notifications
            await this.notifService.createNotification(
                newOwnerId,
                'ownership_transferred',
                'Group Ownership Transferred',
                `You are now the owner of the group`,
                { groupId, oldOwnerId: currentOwnerId }
            );

            await this.notifService.createNotification(
                currentOwnerId,
                'ownership_transferred',
                'Ownership Transferred',
                `You have transferred group ownership`,
                { groupId, newOwnerId }
            );
        } catch (error) {
            await client.query('ROLLBACK');
            throw error;
        } finally {
            client.release();
        }
    }

    /**
     * Get comprehensive group overview including services, subscriptions, and expenses
     */
    async getGroupOverview(userId: string, groupId: string): Promise<any> {
        // Check read permission
        const member = await this.memberRepo.findByGroupAndUser(groupId, userId);
        if (!member) {
            throw new Error('You are not a member of this group');
        }

        const group = await this.groupRepo.findById(groupId);
        const services = await this.groupServiceRepo.findByGroupId(groupId);
        const members = await this.memberRepo.getMembersByGroupId(groupId);

        // Get expenses (assuming you have an ExpenseRepository)
        // const expenses = await this.expenseRepo.findByGroupId(groupId);

        return {
            group,
            services,
            members,
            // expenses,
            summary: {
                totalMembers: members.length,
                activeMembers: members.filter(m => m.joined_at).length,
                totalServices: services.length,
                activeServices: services.filter(s => s.status === 'active').length,
            }
        };
    }
}

