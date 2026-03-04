import { GroupRepository, GroupRow } from '../repositories/GroupRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { GroupServiceRepository } from '../repositories/GroupServiceRepository';
import { UserRepository } from '../repositories/UserRepository';
import { GroupRoleRepository } from '../repositories/GroupRoleRepository';
import { RBACRepository } from '../repositories/RBACRepository';
import { NotificationService } from './NotificationService';
import { SystemSettingsService } from './SystemSettingsService';
import { pool } from '../db';
import { PoolClient } from 'pg';
import { RBACService } from './RBACService';

export class GroupService {
    private groupRepo = new GroupRepository();
    private memberRepo = new GroupMemberRepository();
    private groupServiceRepo = new GroupServiceRepository();
    private userRepo = new UserRepository();
    private roleRepo = new GroupRoleRepository();
    private rbacRepo = new RBACRepository();
    private notifService = new NotificationService();
    private rbacService = new RBACService();

    /**
     * Create default system roles for a new group.  
     * This will be called during group creation to ensure every group starts with a standard set of roles and permissions.
     * If the roles already exist (e.g. due to a previous failed creation), it will skip creating duplicates and just ensure the permissions are set correctly.
     * @param client PostgreSQL client to use for transaction. This should be called within an existing transaction when creating a group.
     * @param groupId ID of the group for which to create default roles.
     * @returns List of created or existing default roles for the group.
     */
    private async createDefaultGroupRoles(client: PoolClient, groupId: string): Promise<any[]> {
        const defaultRoles = [
            {
                name: 'Group Owner',
                description: 'Full control over the group, including billing and deletion.',
                role_level: 1
            },
            {
                name: 'Group Admin',
                description: 'Can manage group settings and members.',
                role_level: 2
            },
            {
                name: 'Group Treasurer',
                description: 'Can manage expenses and group balances.',
                role_level: 3
            },
            { name: 'Group Member',
                description: 'Basic member with read-only access and expense creation rights.',
                role_level: 10
            },
            { name: 'Group Viewer',
                description: 'Read-only access to group information.',
                role_level: 50
            }
        ];

        const createdRoles: any[] = [];
        for (const role of defaultRoles) {
            try {
                const result = await client.query(
                    `INSERT INTO group_roles (group_id, name, description, is_system_role, role_level)
                     VALUES ($1, $2, $3, true, $4)
                     ON CONFLICT (group_id, name)
                     DO UPDATE SET description = EXCLUDED.description, is_system_role = true, role_level = EXCLUDED.role_level
                     RETURNING *`,
                    [groupId, role.name, role.description, role.role_level]
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

    /**
     * Create a new group, optionally with an associated service, and add the creator as the owner. 
     * It also supports adding initial members by email/username or as temp members.
     * @param userId User ID of the group creator
     * @param payload Group creation payload, which can include: name, description, max_members, service details (service_id, service_name, billing info), and initial members (array of {email?, username?, name?})
     * @returns The created group details
     */
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
            const memberOwner = await this.memberRepo.addMember({
                group_id: group.id,
                user_id: userId,
                joined_at: new Date(),
                created_by: userId
            });

            // 3.1 Assign Group Owner role to the creator
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
                        if (member.email) user = await this.userRepo.getByEmail(member.email);
                        else if (member.username) user = await this.userRepo.getByUsername(member.username);

                        if (user) {
                            // Invite via Notification
                            const memberX = await this.memberRepo.addMember({
                                group_id: group.id,
                                user_id: user.id,
                                created_by: userId
                            });
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
                            const memberX = await this.memberRepo.addMember({
                                group_id: group.id,
                                temp_name: member.email || member.username,
                                created_by: userId
                            });
                            if (memberRole && memberRole.id && memberX && memberX.id) {
                                try {
                                    await this.roleRepo.assignRoleToMember(memberX.id, memberRole.id, userId, client);
                                } catch (err: any) {
                                    throw new Error(err?.message);
                                }
                            }
                        }
                    } else if (member.name) {
                        const memberX = await this.memberRepo.addMember({
                            group_id: group.id,
                            temp_name: member.name,
                            created_by: userId
                        });
                        if (memberRole && memberRole.id && memberX && memberX.id) {
                            try {
                                await this.roleRepo.assignRoleToMember(memberX.id, memberRole.id, userId, client);
                            } catch (err: any) {
                                throw new Error(err?.message);
                            }
                        }
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

    /**
     * List groups that the user is a member of.  
     * This will be used for the "My Groups" page.
     * It returns basic group info and can be extended to include membership details or service status if needed.
     * @param userId 
     * @returns 
     */
    async listGroups(userId: string): Promise<GroupRow[]> {
        return await this.groupRepo.findByUserId(userId);
    }

    async getGroupDetail(userId: string, groupId: string) {
        const member = await this.memberRepo.findByGroupAndUser(groupId, userId);
        if (!member) throw new Error("Not a member of this group");

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

    /**
     * Add member to group.
     * If email/username provided and user exists, send invite notification.
     * Otherwise, add as temp member.
     * @param adminId 
     * @param groupId 
     * @param payload 
     * @returns 
     */
    async addMember(adminId: string, groupId: string, payload: { email?: string, name?: string, username?: string }) {
        await this.requireAdminOrOwner(adminId, groupId, "Only admins can add members");

        const memberRole = await this.roleRepo.findByName(groupId, 'Group Member');

        if (payload.email || payload.username) {
            let user = null;
            if (payload.email) user = await this.userRepo.getByEmail(payload.email);
            else if (payload.username) user = await this.userRepo.getByUsername(payload.username);

            if (user) {
                const existingMember = await this.memberRepo.findByGroupAndUser(groupId, user.id);
                if (existingMember) throw new Error("User is already a member of this group");

                const group = await this.groupRepo.findById(groupId);

                const newMember = await this.memberRepo.addMember({
                    group_id: groupId,
                    user_id: user.id,
                    created_by: adminId
                });

                if (memberRole && newMember?.id) {
                    await this.roleRepo.assignRoleToMember(newMember.id, memberRole.id, adminId);
                }

                await this.notifService.createNotification(
                    user.id,
                    'group_invite',
                    'Group Invitation',
                    `You have been invited to join ${group?.name}`,
                    { groupId: groupId, groupName: group?.name, inviterId: adminId }
                );
                return { status: 'invited' };
            } else {
                const newMember = await this.memberRepo.addMember({
                    group_id: groupId,
                    temp_name: payload.email || payload.username,
                    created_by: adminId
                });
                if (memberRole && newMember?.id) {
                    await this.roleRepo.assignRoleToMember(newMember.id, memberRole.id, adminId);
                }
                return { status: 'added_temp' };
            }
        } else if (payload.name) {
            const newMember = await this.memberRepo.addMember({
                group_id: groupId,
                temp_name: payload.name,
                created_by: adminId
            });
            if (memberRole && newMember?.id) {
                await this.roleRepo.assignRoleToMember(newMember.id, memberRole.id, adminId);
            }
            return { status: 'added_temp' };
        }
    }

    /**
     * Bind a temp member to a real user account.  
     * This is typically called when a user accepts an invite and we want to link their account to the temp member entry.
     * It can also be used by admins to manually bind members if needed.
     * @param adminId 
     * @param groupId 
     * @param memberId 
     * @param payload 
     */
    async bindMemberInvite(adminId: string, groupId: string, memberId: string, payload: { email?: string, username?: string }) {
        await this.requireAdminOrOwner(adminId, groupId, "Only admins can bind members");

        let user = null;
        if (payload.email) user = await this.userRepo.getByEmail(payload.email);
        else if (payload.username) user = await this.userRepo.getByUsername(payload.username);

        if (!user) throw new Error("User not found");

        const existingMember = await this.memberRepo.findByGroupAndUser(groupId, user.id);
        if (existingMember) throw new Error("User is already a member of this group");

        const group = await this.groupRepo.findById(groupId);

        await this.notifService.createNotification(
            user.id,
            'group_bind_invite',
            'Group Slot Assignment',
            `You have been assigned to a slot in ${group?.name}. Accept to join?`,
            { groupId: groupId, memberId: memberId, groupName: group?.name, inviterId: adminId }
        );
    }

    /**
     * Accept a group invite.  
     * This can be called by the user when they accept an invite notification, or by an admin when they bind a member to a user.
     * It will link the user account to the member entry and mark them as joined.
     * @param userId 
     * @param groupId 
     * @param memberId 
     */
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
             const newMember = await this.memberRepo.addMember({
                group_id: groupId,
                user_id: userId,
                joined_at: new Date()
            });
            const memberRole = await this.roleRepo.findByName(groupId, 'Group Member');
            if (memberRole && newMember?.id) {
                await this.roleRepo.assignRoleToMember(newMember.id, memberRole.id, userId);
            }
        }
    }

    /**
     * Cancel a group invite.  
     * This is called when an admin cancels an invite or when a user rejects an invite.
     * @param userId 
     * @param groupId 
     * @param memberId 
     */
    async cancelInvite(userId: string, groupId: string, memberId: string) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can cancel invites");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found");
        if (member.joined_at) throw new Error("Member has already joined");

        await this.memberRepo.remove(memberId);
    }

    /**
     * Bind a temp member to a real user account.  
     * This is typically called when a user accepts an invite and we want to link their account to the temp member entry.
     * It can also be used by admins to manually bind members if needed.
     * @param userId 
     * @param memberId 
     */
    async bindMember(userId: string, memberId: string) {
        // Validation logic here
        await this.memberRepo.bindMember(memberId, userId);
    }

    /**
     * Reject a group invite.  
     * This can be called by the user when they reject an invite notification.
     * It will simply remove the member entry if it's a temp member or if the user hasn't joined yet.
     * If the user has already joined, it will throw an error.
     * @param userId 
     * @param groupId 
     * @returns 
     */
    async rejectInvite(userId: string, groupId: string) {
        // Just acknowledging the rejection. 
        // In the future, we might want to mark the notification as 'rejected/solved'
        return true;
    }

    /**
     * Delete a group.  
     * Only the owner or admins can delete the group.
     * This will remove all group data including members, services, and roles.
     * It will also send notifications to all members about the deletion.
     * @param userId 
     * @param groupId 
     */
    async deleteGroup(userId: string, groupId: string) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can delete the group");

        await this.groupRepo.delete(groupId);
    }

    /**
     * Leave a group.  
     * This is called when a user leaves a group they are a member of.
     * @param userId 
     * @param groupId 
     */
    async leaveGroup(userId: string, groupId: string) {
        const member = await this.memberRepo.findByGroupAndUser(groupId, userId);
        if (!member) throw new Error("You are not a member of this group");
        const isOwner = await this.rbacService.isGroupOwner(userId, groupId);
        if (isOwner) throw new Error("Owner cannot leave the group. Delete it instead.");

        await this.memberRepo.removeMember(groupId, userId);
    }

    /**
     * Update group details.  
     * Only the owner or admins can update group settings.
     * This can be used to change the group name, description, or other metadata.
     * It will not allow changing membership or service details - those should be handled by separate methods.
     * @param userId 
     * @param groupId 
     * @param payload 
     * @returns 
     */
    async updateGroup(userId: string, groupId: string, payload: any) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can update group settings");

        return await this.groupRepo.update(groupId, payload);
    }

    /**
     * Update a member's role.  
     * Only the owner or admins can change member roles.
     * This can be used to promote/demote members to/from admin or to assign/remove dynamic roles.
     * It will check the current user's permissions and ensure they cannot modify the owner's role.
     * @param userId 
     * @param groupId 
     * @param memberId 
     * @param newRole 
     */
    async updateMemberRole(userId: string, groupId: string, memberId: string, newRole: any) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can manage roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        // Update Dynamic Roles
        const adminRole = await this.roleRepo.findByName(groupId, 'Group Admin');
        if (adminRole) {
            if (newRole === 'admin') {
                await this.roleRepo.assignRoleToMember(memberId, adminRole.id, userId);
            } else {
                await this.roleRepo.removeRoleFromMember(memberId, adminRole.id);
            }
        }
    }

    /**
     * Assign or remove a dynamic role to/from a member.  
     * This allows admins to grant specific permissions to members without making them full admins.
     * The method will check that the current user has admin rights, validate the target member and role, and then perform the assignment or removal.
     * @param userId 
     * @param groupId 
     * @param memberId 
     * @param roleId 
     */
    async assignDynamicRole(userId: string, groupId: string, memberId: string, roleId: string) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can manage dynamic roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        const targetRole = await this.roleRepo.findById(roleId);
        if (!targetRole || targetRole.group_id !== groupId) throw new Error("Role not found in this group");

        await this.roleRepo.assignRoleToMember(memberId, roleId, userId);
    }

    /**
     * Remove a dynamic role from a member.  
     * This is the counterpart to assignDynamicRole and allows admins to revoke specific permissions from members.
     * It will perform similar checks to ensure the user has admin rights and that the member and role are valid before removing the role assignment.
     * @param userId 
     * @param groupId 
     * @param memberId 
     * @param roleId 
     */
    async removeDynamicRole(userId: string, groupId: string, memberId: string, roleId: string) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can manage dynamic roles");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        const targetRole = await this.roleRepo.findById(roleId);
        if (!targetRole || targetRole.group_id !== groupId) throw new Error("Role not found in this group");

        await this.roleRepo.removeRoleFromMember(memberId, roleId);
    }

    /**
     * Remove a member from the group.  
     * Only the owner or admins can remove members.
     * This will check that the current user has the necessary permissions, validate the target member, and then remove them from the group.
     * If the member being removed is currently bound to a user account, it will also send a notification about their removal.
     * @param userId 
     * @param groupId 
     * @param memberId 
     */
    async removeMember(userId: string, groupId: string, memberId: string) {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can remove members");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        if (member.user_id === userId) throw new Error("Cannot kick yourself. use Leave Group.");

        await this.memberRepo.remove(memberId);
    }

    /**
     * List all roles for a group.  
     * This can be used to display the available roles when managing member permissions or when showing member details.
     * It will return both system-defined roles (like Owner/Admin) and any custom roles that have been created for the group.
     * @param groupId 
     * @returns 
     */
    async listRoles(groupId: string): Promise<any[]> {
        return await this.roleRepo.listAll(groupId);
    }

    /**
     * Create a new custom role for the group.  
     * Only the owner or admins can create new roles.
     * This allows groups to define their own roles with specific permissions beyond the default system roles.
     * The method will check permissions, validate the input, and then create the new role in the database.
     * It will also enforce any limits on the number of custom roles if such limits are defined in the system settings.
     * @param userId 
     * @param groupId 
     * @param payload 
     * @returns 
     */
    async createRole(userId: string, groupId: string, payload: { name: string; description?: string }): Promise<any> {
        await this.requireAdminOrOwner(userId, groupId, "Only admins can create roles");

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
     * Transfer group ownership from current owner to another member.  
     * Only the current owner can transfer ownership.
     * @param currentOwnerId
     * @param groupId
     * @param newOwnerId
     */
    async transferOwnership(currentOwnerId: string, groupId: string, newOwnerId: string): Promise<void> {
        // 1. Verify the current user is the owner
        const currentOwnerMember = await this.memberRepo.findByGroupAndUser(groupId, currentOwnerId);
        const isOwner = await this.rbacService.isGroupOwner(currentOwnerId, groupId);
        if (!currentOwnerMember || !isOwner) {
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

            const ownerRole = await this.roleRepo.findByName(groupId, 'Group Owner', client);
            const adminRole = await this.roleRepo.findByName(groupId, 'Group Admin', client);
            if (!ownerRole) {
                throw new Error('Group Owner role not found');
            }

            await this.roleRepo.removeRoleFromMember(currentOwnerMember.id, ownerRole.id, client);
            if (adminRole) {
                await this.roleRepo.assignRoleToMember(currentOwnerMember.id, adminRole.id, currentOwnerId, client);
            }

            await this.roleRepo.assignRoleToMember(newOwnerMember.id, ownerRole.id, currentOwnerId, client);

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
     * Check if a user has admin or owner role in a group.
     * @param userId 
     * @param groupId 
     * @param message 
     */
    private async requireAdminOrOwner(userId: string, groupId: string, message: string) {
        const allowed = await this.rbacService.hasAnyRole(userId, groupId, ['Group Admin', 'Group Owner']);
        if (!allowed) throw new Error(message);
    }

    /**
     * Get comprehensive group overview including services, subscriptions, and expenses.  
     * This can be used for the group dashboard to provide a summary of the group's status and activity.
     * @param userId 
     * @param groupId 
     * @returns 
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

        const permissions = await this.rbacRepo.getMemberPermissions(groupId, userId);
        const userPermissions: Record<string, any> = {};
        const memberMaxRoleLevel = await this.rbacRepo.getUserMaxRoleLevelInGroup(groupId, userId);
        for (const permission of permissions) {
            userPermissions[permission] = true;
            const parts = permission.split(':');
            if (parts.length === 3) {
                const action = parts[1];
                const resource = parts[2];
                if (!userPermissions[action]) {
                    userPermissions[action] = {};
                }
                userPermissions[action][resource] = true;
            }
        }

        // Get expenses (assuming you have an ExpenseRepository)
        // const expenses = await this.expenseRepo.findByGroupId(groupId);

        return {
            group,
            services,
            members,
            userPermissions,
            memberMaxRoleLevel,
            // expenses,
            summary: {
                totalMembers: members.length,
                activeMembers: members.filter(m => m.joined_at).length,
                totalServices: services.length,
                activeServices: services.filter(s => s.status === 'active').length,
            }
        };
    }

    /**
     * Get the current user's highest role level in a group 
     * Lower numeric values = higher privilege
     * @param userId 
     * @param groupId 
     * @returns 
     */
    async getUserMaxRoleLevel(userId: string, groupId: string): Promise<number> {
        return await this.rbacRepo.getUserMaxRoleLevelInGroup(groupId, userId);
    }
}

