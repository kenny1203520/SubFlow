import cron from 'node-cron';
import { pool } from './db';

export class SchedulerService {
    static init() {
        console.log('Initializing Scheduler Service...');

        // Run every day at 00:00 (midnight)
        cron.schedule('0 0 * * *', async () => {
            console.log('Running daily billing check...');
            await this.generateBills();
        });

        console.log('Scheduler Service started.');
    }

    private static async generateBills() {
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

            console.log(`Found ${dueGroups.rows.length} groups due for billing.`);

            for (const group of dueGroups.rows) {
                // 1. Create a new Bill
                const issueDate = new Date();
                const dueDate = new Date();
                dueDate.setDate(dueDate.getDate() + 7); // Due in 7 days by default

                const billRes = await client.query(`
                    INSERT INTO bills (
                        group_id, title, description, total_amount, currency, 
                        issue_date, due_date, status, created_by
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                    RETURNING id
                `, [
                    group.id,
                    `${group.name} - ${issueDate.toLocaleDateString()}`,
                    `Automated bill for ${group.billing_cycle} subscription`,
                    group.amount,
                    group.currency,
                    issueDate,
                    dueDate,
                    'pending',
                    group.created_by // System generated, but attributed to creator/admin
                ]);

                const billId = billRes.rows[0].id;

                // 2. Calculate Splits
                const membersRes = await client.query(`
                    SELECT * FROM group_members WHERE group_id = $1
                `, [group.id]);

                const members = membersRes.rows;
                if (members.length === 0) continue;

                let splitAmount = 0;

                // For now, support 'equal' split only as per immediate requirement, 
                // but structure allows for 'fixed' or 'percentage' extension.
                if (group.billing_method === 'equal') {
                    splitAmount = parseFloat((group.amount / members.length).toFixed(2));
                }
                // TODO: Implement other billing methods

                // 3. Create Bill Splits
                for (const member of members) {
                    await client.query(`
                        INSERT INTO bill_splits (
                            bill_id, member_id, user_id, amount_owed, status
                        ) VALUES ($1, $2, $3, $4, $5)
                    `, [
                        billId,
                        member.id,
                        member.user_id, // Can be null
                        splitAmount,
                        'pending'
                    ]);
                }

                // 4. Update Group Next Payment Date
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

                // 5. Audit Log
                await client.query(`
                    INSERT INTO audit_logs (
                        target_type, target_id, action, changes
                    ) VALUES ($1, $2, $3, $4)
                `, [
                    'group',
                    group.id,
                    'auto_billing',
                    JSON.stringify({ bill_id: billId, amount: group.amount })
                ]);
            }

            await client.query('COMMIT');
            console.log('Daily billing check completed successfully.');
        } catch (error) {
            await client.query('ROLLBACK');
            console.error('Error in billing scheduler:', error);
        } finally {
            client.release();
        }
    }
}
