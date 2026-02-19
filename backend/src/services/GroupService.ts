import { GroupRepository, GroupRow } from '../repositories/GroupRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { UserRepository } from '../repositories/UserRepository';
import { NotificationService } from './NotificationService';
import { pool } from '../db';

export class GroupService {
    private groupRepo = new GroupRepository();
    private memberRepo = new GroupMemberRepository();
    private userRepo = new UserRepository();
    private notifService = new NotificationService();

    async createGroup(userId: string, payload: any): Promise<GroupRow> {
        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            // Handle Icon & Service (Simplified for now)
            let iconUrl = '';
            if (payload.website) {
                try {
                    const domain = new URL(payload.website).hostname;
                    iconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=64`;
                } catch (e) { }
            }

            // In a real OOP refactor, we would have a ServiceRepository too
            // For now, let's keep it in the transaction
            let serviceId = null;
            if (payload.service_name) {
                const serviceRes = await client.query("SELECT id FROM services WHERE name = $1", [payload.service_name]);
                if (serviceRes.rows.length > 0) {
                    serviceId = serviceRes.rows[0].id;
                } else {
                    const newService = await client.query(
                        "INSERT INTO services (name, domain, icon_url, created_by) VALUES ($1, $2, $3, $4) RETURNING id",
                        [payload.service_name, payload.website || '', iconUrl, userId]
                    );
                    serviceId = newService.rows[0].id;
                }
            }

            // Sanitize Date Fields
            const sanitizeDate = (d: any) => (d && d !== '') ? new Date(d) : null;
            const sanitizeString = (s: any) => (s && s !== '') ? s : null;

            const group = await this.groupRepo.create({
                ...payload,
                service_id: serviceId,
                created_by: userId,
                next_payment_date: sanitizeDate(payload.next_payment_date),
                start_date: sanitizeDate(payload.start_date),
                end_value: (payload.end_condition === 'date') ? (sanitizeDate(payload.end_value)?.toISOString() || null) : sanitizeString(payload.end_value)
            });

            await this.memberRepo.addMember({
                group_id: group.id,
                user_id: userId,
                role: 'admin'
            });

            if (payload.initial_members) {
                for (const member of payload.initial_members) {
                    if (member.email || member.username) {
                        let user = null;
                        if (member.email) user = await this.userRepo.findByEmail(member.email);
                        else if (member.username) user = await this.userRepo.findByUsername(member.username);

                        if (user) {
                            // Invite via Notification
                            await this.notifService.createNotification(
                                user.id,
                                'group_invite',
                                'Group Invitation',
                                `You have been invited to join ${group.name}`,
                                { groupId: group.id, groupName: group.name, inviterId: userId }
                            );
                        } else {
                            // Fallback to temp member if user not found (or treat as error? For now fallback)
                            // User wanted "email and username". 
                            await this.memberRepo.addMember({
                                group_id: group.id,
                                temp_name: member.email || member.username,
                                role: 'member'
                            });
                        }
                    } else if (member.name) {
                        await this.memberRepo.addMember({
                            group_id: group.id,
                            temp_name: member.name,
                            role: 'member'
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
        const members = await this.memberRepo.getMembersByGroupId(groupId);

        return { group, members };
    }

    async addMember(adminId: string, groupId: string, payload: { email?: string, name?: string, username?: string }) {
        const role = await this.memberRepo.checkRole(groupId, adminId);
        if (role !== 'admin') throw new Error("Only admins can add members");

        // Logic split: Invite vs Temp
        if (payload.email || payload.username) {
            let user = null;
            if (payload.email) user = await this.userRepo.findByEmail(payload.email);
            else if (payload.username) user = await this.userRepo.findByUsername(payload.username);

            if (user) {
                // Check if already member
                const existingRole = await this.memberRepo.checkRole(groupId, user.id);
                if (existingRole) throw new Error("User is already a member of this group");

                const group = await this.groupRepo.findById(groupId);

                // Check if already invited (optional, depends on notifRepo capabilities, but good practice)
                // For now, let's assume sending another notification is okay (bump), or we can check notifications table

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
                    role: 'member'
                });
                return { status: 'added_temp' };
            }
        } else if (payload.name) {
            await this.memberRepo.addMember({
                group_id: groupId,
                temp_name: payload.name,
                role: 'member'
            });
            return { status: 'added_temp' };
        }
    }

    async inviteUserToBind(adminId: string, memberId: string, payload: { email?: string, username?: string }) {
        const member = (await this.memberRepo.getMembersByGroupId(memberId))[0]; // This logic is wrong, need findById
        // memberId is the row ID in group_members
        // We need to fetch the member row first to get groupId
        // But Repository needs findById
        // Let's assume we pass groupId for permission check or fetch it.

        // Actually, let's fix this properly. We need `findById` in GroupMemberRepository
        throw new Error("Implementation Pending: Need findById in MemberRepo");
    }

    // Fixed inviteUserToBind (Implementation below assumes repo update)
    async bindMemberInvite(adminId: string, groupId: string, memberId: string, payload: { email?: string, username?: string }) {
        const role = await this.memberRepo.checkRole(groupId, adminId);
        if (role !== 'admin') throw new Error("Only admins can bind members");

        let user = null;
        if (payload.email) user = await this.userRepo.findByEmail(payload.email);
        else if (payload.username) user = await this.userRepo.findByUsername(payload.username);

        if (!user) throw new Error("User not found");

        // Check if already member
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
        // Check if already member
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role) throw new Error("Already a member");

        if (memberId) {
            // Binding Logic
            await this.memberRepo.bindMember(memberId, userId);
        } else {
            // New Member Logic
            await this.memberRepo.addMember({
                group_id: groupId,
                user_id: userId,
                role: 'member'
            });
        }
    }

    async rejectInvite(userId: string, groupId: string) {
        // Just acknowledging the rejection. 
        // In a real system we might log this or notify the inviter.
        // For now, the notification is marked read by the controller/frontend.
        return true;
    }

    async bindMember(userId: string, memberId: string) {
        // Validation logic here
        await this.memberRepo.bindMember(memberId, userId);
    }

    async deleteGroup(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin') throw new Error("Only admins can delete the group");

        // Transaction/Cascade delete is handled by DB usually, or we can explicit delete
        await this.groupRepo.delete(groupId);
    }

    async leaveGroup(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("You are not a member of this group");
        if (role === 'admin') throw new Error("Admins cannot leave the group. Delete it instead.");

        await this.memberRepo.removeMember(groupId, userId);
    }

    async updateGroup(userId: string, groupId: string, payload: any) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin') throw new Error("Only admins can update group settings");

        // Sanitize Date Fields
        const sanitizeDate = (d: any) => (d && d !== '') ? new Date(d) : null;
        const sanitizeString = (s: any) => (s && s !== '') ? s : null;

        const updateData: any = {
            ...payload,
            next_payment_date: sanitizeDate(payload.next_payment_date),
            start_date: sanitizeDate(payload.start_date),
            end_value: (payload.end_condition === 'date') ? (sanitizeDate(payload.end_value)?.toISOString() || null) : sanitizeString(payload.end_value)
        };

        // If service name changed, might need to handle service_id logic, but for now just update fields
        // In a full implementation, we'd check if service exists, etc.
        // For now, let's assume direct update of group fields

        return await this.groupRepo.update(groupId, updateData);
    }

    async updateMemberRole(userId: string, groupId: string, memberId: string, newRole: 'admin' | 'member') {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin') throw new Error("Only admins can manage roles");

        // Prevent demoting self if only admin? 
        // For now, simple logic.

        // Helper to check if member belongs to group 
        // (We should probable add findById(memberId) to check group_id, but trusting the frontend/logic for now with a verify)
        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        await this.memberRepo.updateRole(memberId, newRole);
    }

    async removeMember(userId: string, groupId: string, memberId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (role !== 'admin') throw new Error("Only admins can remove members");

        const member = await this.memberRepo.findById(memberId);
        if (!member || member.group_id !== groupId) throw new Error("Member not found in this group");

        if (member.user_id === userId) throw new Error("Cannot kick yourself. use Leave Group.");

        // TODO: Check debts?

        await this.memberRepo.remove(memberId);
    }
}