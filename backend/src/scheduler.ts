import cron from 'node-cron';
import { BillService } from './services/BillService';
import { GroupRepository } from './repositories/GroupRepository';
import { pool } from './db';

export class SchedulerService {
    static init() {
        // console.log('Initializing Scheduler Service...');

        // Run every day at 00:00 (midnight)
        cron.schedule('0 0 * * *', async () => {
            // console.log('Running daily billing check...');
            await this.generateBills();
        });

        // console.log('Scheduler Service started.');
    }

    private static async generateBills() {
        // We still need raw query to find due groups efficiently, 
        // or add a method to GroupRepository.
        // Let's add the query here for now, but use BillService for creation.

        const billService = new BillService();
        const client = await pool.connect();

        try {
            await client.query('BEGIN');

            // Find active groups due for payment today or earlier
            const dueGroups = await client.query(`
                SELECT * FROM groups 
                WHERE status = 'active' 
                AND next_payment_date IS NOT NULL 
                AND next_payment_date <= CURRENT_DATE
                AND billing_cycle IN ('monthly', 'yearly')
            `);

            // console.log(`Found ${dueGroups.rows.length} groups due for billing.`);

            for (const group of dueGroups.rows) {
                const issueDate = new Date();
                const dueDate = new Date();
                dueDate.setDate(dueDate.getDate() + 7);

                try {
                    // Use BillService to create the bill and splits
                    // Note: BillService.createBill expects { groupId, title, description, amount, splits: [...] }
                    // We need to fetch members to calculate splits first.

                    // Actually, let's keep the logic here but reuse BillService if possible.
                    // Since BillService.createBill takes explicit splits, we should calculate them here.
                    // Or better, extend BillService to have `generateAutoBill(groupId)`?
                    // For this refactor, let's just use BillService.createBill to ensure consistency in bill creation.

                    // Helper to get members
                    const membersRes = await client.query(`SELECT * FROM group_members WHERE group_id = $1`, [group.id]);
                    const members = membersRes.rows;
                    if (members.length === 0) continue;

                    let splitAmount = 0;
                    if (group.billing_method === 'equal') {
                        splitAmount = parseFloat((group.amount / members.length).toFixed(2));
                    }

                    const splits = members.map((m: any) => ({
                        userId: m.user_id, // Note: BillService expects userId not memberId in the interface we defined? 
                        // Let's check BillService.createBill signature.
                        // It takes { userId, amount }. 
                        amount: splitAmount
                    }));

                    // We need a userId for 'createdBy'. System generated? 
                    // Let's use group.created_by or a system ID.
                    const creatorId = group.created_by;

                    await billService.createBill(creatorId, {
                        groupId: group.id,
                        title: `${group.name} - ${issueDate.toLocaleDateString()}`,
                        description: `Automated bill for ${group.billing_cycle} subscription`,
                        amount: group.amount,
                        currency: group.currency,
                        dueDate: dueDate.toISOString(),
                        splits: splits
                    });

                    // Update Group Next Payment Date
                    let nextDate = new Date(group.next_payment_date);
                    if (group.billing_cycle === 'monthly') {
                        nextDate.setMonth(nextDate.getMonth() + 1);
                    } else if (group.billing_cycle === 'yearly') {
                        nextDate.setFullYear(nextDate.getFullYear() + 1);
                    }

                    await client.query(`
                        UPDATE groups 
                        SET next_payment_date = $1, updated_at = NOW()
                        WHERE id = $2
                    `, [nextDate, group.id]);

                } catch (err) {
                    console.error(`Failed to generate bill for group ${group.id}:`, err);
                }
            }

            await client.query('COMMIT');
            // console.log('Daily billing check completed successfully.');
        } catch (error) {
            await client.query('ROLLBACK');
            console.error('Error in billing scheduler:', error);
        } finally {
            client.release();
        }
    }
}

