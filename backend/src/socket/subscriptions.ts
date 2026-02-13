import { Server, Socket } from "socket.io";
import { pool } from "../db";

export const registerSubscriptionHandlers = (io: Server, socket: Socket) => {
    socket.on("subscription:add", async (payload: { groupId: string, name: string, amount: number, billingCycle: string, startDate: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;
            const res = await pool.query(
                "INSERT INTO subscriptions (group_id, name, amount, billing_cycle, start_date) VALUES ($1, $2, $3, $4, $5) RETURNING *",
                [payload.groupId, payload.name, payload.amount, payload.billingCycle, payload.startDate]
            );
            cb({ status: "ok", subscription: res.rows[0] });
        } catch (error) {
            console.error("Error adding subscription:", error);
            cb({ status: "error", message: "Failed to add subscription" });
        }
    });

    socket.on("subscription:list", async (payload: { groupId: string }, cb: (res: any) => void) => {
        try {
            const res = await pool.query(
                "SELECT * FROM subscriptions WHERE group_id = $1",
                [payload.groupId]
            );
            cb({ status: "ok", subscriptions: res.rows });
        } catch (error) {
            console.error("Error listing subscriptions:", error);
            cb({ status: "error", message: "Failed to list subscriptions" });
        }
    });

    socket.on("subscription:all", async (cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;
            const res = await pool.query(
                `SELECT s.* FROM subscriptions s
                 JOIN group_members gm ON s.group_id = gm.group_id
                 WHERE gm.user_id = $1`,
                [userId]
            );
            cb({ status: "ok", subscriptions: res.rows });
        } catch (error) {
            console.error("Error listing all subscriptions:", error);
            cb({ status: "error", message: "Failed to list subscriptions" });
        }
    });
};
