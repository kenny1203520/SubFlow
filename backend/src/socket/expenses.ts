import { Server, Socket } from "socket.io";
import { pool } from "../db";

export const registerExpenseHandlers = (io: Server, socket: Socket) => {
    socket.on("expense:add", async (payload: { groupId: string, amount: number, description: string, splits: { userId: string, amount: number }[] }, cb: (res: any) => void) => {
        try {
            const payerId = socket.data.user.id;
            const client = await pool.connect();
            try {
                await client.query("BEGIN");

                const expenseRes = await client.query(
                    "INSERT INTO expenses (group_id, paid_by, amount, description) VALUES ($1, $2, $3, $4) RETURNING *",
                    [payload.groupId, payerId, payload.amount, payload.description]
                );
                const expense = expenseRes.rows[0];

                for (const split of payload.splits) {
                    await client.query(
                        "INSERT INTO expense_splits (expense_id, user_id, amount_owed) VALUES ($1, $2, $3)",
                        [expense.id, split.userId, split.amount]
                    );
                }

                await client.query("COMMIT");
                cb({ status: "ok", expense });
            } catch (error) {
                await client.query("ROLLBACK");
                console.error("Error adding expense:", error);
                cb({ status: "error", message: "Failed to add expense" });
            } finally {
                client.release();
            }
        } catch (error) {
            console.error("Database connection error:", error);
            cb({ status: "error", message: "Database connection error" });
        }
    });

    socket.on("expense:list", async (payload: { groupId: string }, cb: (res: any) => void) => {
        try {
            const res = await pool.query(
                "SELECT * FROM expenses WHERE group_id = $1 ORDER BY date DESC",
                [payload.groupId]
            );
            cb({ status: "ok", expenses: res.rows });
        } catch (error) {
            console.error("Error listing expenses:", error);
            cb({ status: "error", message: "Failed to list expenses" });
        }
    });

    socket.on("expense:get_splits", async (payload: { groupId: string }, cb: (res: any) => void) => {
        try {
            const res = await pool.query(
                `SELECT es.*, e.description, e.date, u.username as payer_name 
                 FROM expense_splits es
                 JOIN expenses e ON es.expense_id = e.id
                 JOIN users u ON e.paid_by = u.id
                 WHERE e.group_id = $1 AND es.is_paid = false`,
                [payload.groupId]
            );
            cb({ status: "ok", splits: res.rows });
        } catch (error) {
            console.error("Error getting splits:", error);
            cb({ status: "error", message: "Failed to get splits" });
        }
    });

    socket.on("expense:settle", async (payload: { expenseId: string, userId: string }, cb: (res: any) => void) => {
        try {
            // Only the person who owes can mark it as paid, or the admin? 
            // For now, let local user settle their own debt or admin settle any.
            const updaterId = socket.data.user.id;

            await pool.query(
                "UPDATE expense_splits SET is_paid = true WHERE expense_id = $1 AND user_id = $2",
                [payload.expenseId, payload.userId]
            );

            cb({ status: "ok" });
        } catch (error) {
            console.error("Error settling expense:", error);
            cb({ status: "error", message: "Failed to settle expense" });
        }
    });
};
