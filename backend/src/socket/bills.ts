import { Server, Socket } from "socket.io";
import { pool } from "../db";

export const registerBillHandlers = (io: Server, socket: Socket) => {
    // List bills for a group
    socket.on("bill:list", async (payload: { groupId: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Check membership
            const memberCheck = await pool.query(
                "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2",
                [payload.groupId, userId]
            );

            if (memberCheck.rows.length === 0) {
                return cb({ status: "error", message: "Not a member" });
            }

            const billsRes = await pool.query(
                `SELECT * FROM bills WHERE group_id = $1 ORDER BY issue_date DESC`,
                [payload.groupId]
            );

            cb({ status: "ok", bills: billsRes.rows });
        } catch (error) {
            console.error("Error listing bills:", error);
            cb({ status: "error", message: "Failed to list bills" });
        }
    });

    // Get specific bill details and splits
    socket.on("bill:get", async (payload: { billId: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Get bill and verify access (via group membership)
            const billRes = await pool.query(`
                SELECT b.*, g.name as group_name, g.currency 
                FROM bills b
                JOIN groups g ON b.group_id = g.id
                WHERE b.id = $1
            `, [payload.billId]);

            if (billRes.rows.length === 0) {
                return cb({ status: "error", message: "Bill not found" });
            }

            const bill = billRes.rows[0];

            // Check membership
            const memberCheck = await pool.query(
                "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2",
                [bill.group_id, userId]
            );

            if (memberCheck.rows.length === 0) {
                return cb({ status: "error", message: "Not a member" });
            }

            // Get splits with user details
            const splitsRes = await pool.query(`
                SELECT bs.*, 
                       gm.temp_name, 
                       u.username, u.email, u.avatar_url
                FROM bill_splits bs
                JOIN group_members gm ON bs.member_id = gm.id
                LEFT JOIN users u ON gm.user_id = u.id
                WHERE bs.bill_id = $1
            `, [payload.billId]);

            cb({ status: "ok", bill, splits: splitsRes.rows });
        } catch (error) {
            console.error("Error getting bill:", error);
            cb({ status: "error", message: "Failed to fetch bill details" });
        }
    });

    // Update a split (Admin/Host only)
    socket.on("bill:update_split", async (payload: { splitId: string, amount: number }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;
            const client = await pool.connect();

            try {
                await client.query("BEGIN");

                // Verify admin/host permission
                const splitRes = await client.query(`
                    SELECT b.group_id 
                    FROM bill_splits bs
                    JOIN bills b ON bs.bill_id = b.id
                    WHERE bs.id = $1
                `, [payload.splitId]);

                if (splitRes.rows.length === 0) {
                    return cb({ status: "error", message: "Split not found" });
                }

                const groupId = splitRes.rows[0].group_id;

                const adminCheck = await client.query(
                    "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2 AND role = 'admin'",
                    [groupId, userId]
                );

                if (adminCheck.rows.length === 0) {
                    return cb({ status: "error", message: "Only admins can edit bills" });
                }

                // Update split
                const oldSplit = await client.query("SELECT * FROM bill_splits WHERE id = $1", [payload.splitId]);
                await client.query(
                    "UPDATE bill_splits SET amount_owed = $1, updated_at = NOW() WHERE id = $2",
                    [payload.amount, payload.splitId]
                );

                // Log audit
                await client.query(`
                    INSERT INTO audit_logs (
                        user_id, target_type, target_id, action, changes
                    ) VALUES ($1, $2, $3, $4, $5)
                `, [
                    userId,
                    'bill_split',
                    payload.splitId,
                    'update_amount',
                    JSON.stringify({ old: oldSplit.rows[0].amount_owed, new: payload.amount })
                ]);

                // Recalculate Bill Total? 
                // Does updating a split change the total bill amount? 
                // Usually yes, the bill total is sum of splits. Or bill total is fixed and splits must sum to it?
                // For flexible billing, updating a split *should* update the bill total or warn if mismatch.
                // Let's update bill total for now to keep it consistent.

                await client.query(`
                    UPDATE bills 
                    SET total_amount = (SELECT SUM(amount_owed) FROM bill_splits WHERE bill_id = (SELECT bill_id FROM bill_splits WHERE id = $1)),
                        updated_by = $2,
                        updated_at = NOW()
                    WHERE id = (SELECT bill_id FROM bill_splits WHERE id = $1)
                `, [payload.splitId, userId]);

                await client.query("COMMIT");
                cb({ status: "ok" });
            } catch (error) {
                await client.query("ROLLBACK");
                throw error;
            } finally {
                client.release();
            }
        } catch (error) {
            console.error("Error updating split:", error);
            cb({ status: "error", message: "Failed to update split" });
        }
    });
};
