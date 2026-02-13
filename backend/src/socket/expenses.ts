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
};
