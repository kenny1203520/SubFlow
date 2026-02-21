import express from "express";
import { Router } from "express";
import { pool } from "../db";
import { requireAuth, requirePermission } from "../middleware/auth";
import { generateId } from "lucia";

const router = Router();

router.use(requireAuth);

interface Split {
    userId: string;
    amount: number;
}

// Add an expense
router.post("/", requirePermission('groups', 'create', 'expenses'), addExpense);

// Get expenses for a group
router.get("/group/:groupId", requirePermission('groups', 'read', 'expenses'), getExpensesByGroupId);

async function addExpense(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    const { groupId, amount, description, splits, date } = req.body;

    if (!groupId || !amount || !splits || !Array.isArray(splits) || splits.length === 0) {
        return res.status(400).send("Invalid input");
    }

    const client = await pool.connect();
    try {
        await client.query("BEGIN");

        // Verify membership
        const memberCheck = await client.query(
            "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        if (memberCheck.rows.length === 0) {
            await client.query("ROLLBACK");
            return res.status(403).send("Not a member of this group");
        }

        const expenseId = generateId(15);
        await client.query(
            "INSERT INTO expenses (id, group_id, paid_by, amount, description, date) VALUES ($1, $2, $3, $4, $5, $6)",
            [expenseId, groupId, userId, amount, description, date || new Date()]
        );

        for (const split of splits as Split[]) {
            await client.query(
                "INSERT INTO expense_splits (expense_id, user_id, amount_owed) VALUES ($1, $2, $3)",
                [expenseId, split.userId, split.amount]
            );
        }

        await client.query("COMMIT");
        res.status(201).json({ id: expenseId, message: "Expense added" });
    } catch (error) {
        await client.query("ROLLBACK");
        console.error(error);
        res.status(500).send("Server Error");
    } finally {
        client.release();
    }
}

async function getExpensesByGroupId(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    const { groupId } = req.params;

    try {
        const memberCheck = await pool.query(
            "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        if (memberCheck.rows.length === 0) {
            return res.status(403).send("Not a member of this group");
        }

        const result = await pool.query(`
            SELECT e.*, u.username as paid_by_username,
            (SELECT json_agg(json_build_object('user_id', es.user_id, 'amount_owed', es.amount_owed, 'is_paid', es.is_paid, 'username', u2.username))
             FROM expense_splits es
             JOIN users u2 ON es.user_id = u2.id
             WHERE es.expense_id = e.id
            ) as splits
            FROM expenses e
            JOIN users u ON e.paid_by = u.id
            WHERE e.group_id = $1
            ORDER BY e.date DESC
        `, [groupId]);

        res.json(result.rows);
    } catch (error) {
        console.error(error);
        res.status(500).send("Server Error");
    }
}

export default router;
