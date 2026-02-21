import express from 'express';
import { Parser } from 'json2csv';
import { pool } from '../db';
import { requireAuth } from '../middleware/auth';

const router = express.Router();
router.use(requireAuth);

// Export Expenses
router.get('/group/:groupId/expenses', exportExpensesHandler);

// Export Bills
router.get('/group/:groupId/bills', exportBillsHandler);

async function exportExpensesHandler(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    const { groupId } = req.params;

    try {
        // Verify membership
        const memberCheck = await pool.query(
            "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        if (memberCheck.rows.length === 0) {
            return res.status(403).send("Not a member of this group");
        }

        const result = await pool.query(`
             SELECT 
                to_char(e.date, 'YYYY-MM-DD') as Date,
                e.description as Description,
                e.amount as Amount,
                u.username as "Paid By",
                (SELECT string_agg(u2.username || ': ' || es.amount_owed, ' | ')
                 FROM expense_splits es
                 JOIN users u2 ON es.user_id = u2.id
                 WHERE es.expense_id = e.id) as "Split Details"
            FROM expenses e
            JOIN users u ON e.paid_by = u.id
            WHERE e.group_id = $1
            ORDER BY e.date DESC
        `, [groupId]);

        const fields = ['Date', 'Description', 'Amount', 'Paid By', 'Split Details'];
        const json2csvParser = new Parser({ fields });
        const csv = json2csvParser.parse(result.rows);

        res.header('Content-Type', 'text/csv');
        res.attachment(`expenses-${groupId}.csv`);
        res.send(csv);

    } catch (error) {
        console.error("Export Expenses Error:", error);
        res.status(500).send("Server Error");
    }
}

async function exportBillsHandler(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    const { groupId } = req.params;

    try {
        // Verify membership
        const memberCheck = await pool.query(
            "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        if (memberCheck.rows.length === 0) {
            return res.status(403).send("Not a member of this group");
        }

        const result = await pool.query(`
            SELECT 
                b.title as Title,
                b.currency as Currency,
                b.total_amount as "Total Amount",
                to_char(b.issue_date, 'YYYY-MM-DD') as "Issue Date",
                to_char(b.due_date, 'YYYY-MM-DD') as "Due Date",
                b.status as Status,
                u.username as "Created By"
            FROM bills b
            JOIN users u ON b.created_by = u.id
            WHERE b.group_id = $1
            ORDER BY b.issue_date DESC
        `, [groupId]);

        const fields = ['Title', 'Currency', 'Total Amount', 'Issue Date', 'Due Date', 'Status', 'Created By'];
        const json2csvParser = new Parser({ fields });
        const csv = json2csvParser.parse(result.rows);

        res.header('Content-Type', 'text/csv');
        res.attachment(`bills-${groupId}.csv`);
        res.send(csv);

    } catch (error) {
        console.error("Export Bills Error:", error);
        res.status(500).send("Server Error");
    }
}

export default router;
