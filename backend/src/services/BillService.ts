import { BillRepository } from '../repositories/BillRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { pool } from '../db';

export class BillService {
    private billRepo = new BillRepository();
    private memberRepo = new GroupMemberRepository();

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
