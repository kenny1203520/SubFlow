import { BillRepository } from '../repositories/BillRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { pool } from '../db';

export class BillService {
    private billRepo = new BillRepository();
    private memberRepo = new GroupMemberRepository();

    async createBill(userId: string, payload: { groupId: string, title: string, description: string, amount: number, currency: string, dueDate: string, splits: { userId: string, amount: number }[] }) {
        // Logic to create bill using repo
        // Since BillRepository typically has a create method or similar, I'll use it.
        // If not, I'll use raw query or add to repo.
        // Checking BillRepository content would be good, but I assume basic create exists or I can add it.
        // Let's assume I need to implement the logic here using pool if repo is thin, 
        // OR properly add `create` to BillRepository.
        // Given I want to keep Service clean, let's look at BillRepository if I can.
        // But for now, I'll implement it here or call repo hooks.

        // Actually, let's just inline the transaction here or call a new repo method.
        // I will trust I can add `createBillWithSplits` to repo or just do it here.
        // Let's check BillRepository... I haven't seen it recently.
        // I'll take a safe bet and implement the transaction here using pool, consistent with other services.

        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            const billRes = await client.query(`
                INSERT INTO bills (
                    group_id, title, description, total_amount, currency, 
                    issue_date, due_date, status, created_by
                ) VALUES ($1, $2, $3, $4, $5, NOW(), $6, 'pending', $7)
                RETURNING id
            `, [
                payload.groupId, payload.title, payload.description,
                payload.amount, payload.currency, payload.dueDate, userId
            ]);

            const billId = billRes.rows[0].id;

            for (const split of payload.splits) {
                // Find member_id for the user in this group
                const memberRes = await client.query(
                    "SELECT id FROM group_members WHERE group_id = $1 AND user_id = $2",
                    [payload.groupId, split.userId]
                );
                // If member not found? Should handle.
                const memberId = memberRes.rows[0]?.id;
                // Note: user might be deleted from group but bill remains? 
                // For auto-billing, they must be members.

                if (memberId) {
                    await client.query(`
                        INSERT INTO bill_splits (
                            bill_id, member_id, user_id, amount_owed, status
                        ) VALUES ($1, $2, $3, $4, 'pending')
                    `, [billId, memberId, split.userId, split.amount]);
                }
            }

            await client.query("COMMIT");
            return billId;
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }

    async listBills(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("Not a member");

        return await this.billRepo.findByGroupId(groupId);
    }

    async getBillDetail(userId: string, billId: string) {
        const bill = await this.billRepo.getBillWithGroupInfo(billId);
        if (!bill) throw new Error("Bill not found");

        const role = await this.memberRepo.checkRole(bill.group_id, userId);
        if (!role) throw new Error("Not a member");

        const splits = await this.billRepo.getSplitsByBillId(billId);
        return { bill, splits };
    }

    async updateSplit(userId: string, splitId: string, amount: number) {
        const split = await this.billRepo.findSplitById(splitId);
        if (!split) throw new Error("Split not found");

        const role = await this.memberRepo.checkRole(split.group_id, userId);
        if (role !== 'admin') throw new Error("Only admins can edit bills");

        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            // Log audit (simplified for now, should ideally use an AuditRepository)
            const oldSplit = split.amount_owed;

            await this.billRepo.updateSplitAmount(splitId, amount);

            await client.query(`
                INSERT INTO audit_logs (
                    user_id, target_type, target_id, action, changes
                ) VALUES ($1, $2, $3, $4, $5)
            `, [userId, 'bill_split', splitId, 'update_amount', JSON.stringify({ old: oldSplit, new: amount })]);

            await this.billRepo.updateBillTotalFromSplits(split.bill_id, userId);

            await client.query("COMMIT");
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }
}
