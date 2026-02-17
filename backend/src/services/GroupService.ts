import { GroupRepository, GroupRow } from '../repositories/GroupRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { UserRepository } from '../repositories/UserRepository';
import { pool } from '../db';

export class GroupService {
    private groupRepo = new GroupRepository();
    private memberRepo = new GroupMemberRepository();
    private userRepo = new UserRepository();

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

            const group = await this.groupRepo.create({
                ...payload,
                service_id: serviceId,
                created_by: userId,
                next_payment_date: payload.next_payment_date ? new Date(payload.next_payment_date) : null
            });

            await this.memberRepo.addMember({
                group_id: group.id,
                user_id: userId,
                role: 'admin'
            });

            if (payload.initial_members) {
                for (const member of payload.initial_members) {
                    if (member.email) {
                        const user = await this.userRepo.findByEmail(member.email);
                        await this.memberRepo.addMember({
                            group_id: group.id,
                            user_id: user?.id,
                            temp_name: user ? undefined : member.email,
                            role: 'member'
                        });
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

    async addMember(adminId: string, groupId: string, payload: { email?: string, name?: string }) {
        const role = await this.memberRepo.checkRole(groupId, adminId);
        if (role !== 'admin') throw new Error("Only admins can add members");

        if (payload.email) {
            const user = await this.userRepo.findByEmail(payload.email);
            await this.memberRepo.addMember({
                group_id: groupId,
                user_id: user?.id,
                temp_name: user ? undefined : payload.email,
                role: 'member'
            });
        } else if (payload.name) {
            await this.memberRepo.addMember({
                group_id: groupId,
                temp_name: payload.name,
                role: 'member'
            });
        }
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

 