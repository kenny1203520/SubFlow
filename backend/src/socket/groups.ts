import { Server, Socket } from "socket.io";
import { pool } from "../db";

export const registerGroupHandlers = (io: Server, socket: Socket) => {
    socket.on("group:create", async (payload: { name: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;
            const client = await pool.connect();
            try {
                await client.query("BEGIN");

                const groupRes = await client.query(
                    "INSERT INTO groups (name, created_by) VALUES ($1, $2) RETURNING *",
                    [payload.name, userId]
                );
                const group = groupRes.rows[0];

                await client.query(
                    "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)",
                    [group.id, userId, 'admin']
                );

                await client.query("COMMIT");

                cb({ status: "ok", group });
            } catch (error) {
                await client.query("ROLLBACK");
                console.error("Error creating group:", error);
                cb({ status: "error", message: "Failed to create group" });
            } finally {
                client.release();
            }
        } catch (error) {
            console.error("Database connection error:", error);
            cb({ status: "error", message: "Database connection error" });
        }
    });

    socket.on("group:list", async (cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;
            const res = await pool.query(
                `SELECT g.* FROM groups g 
                 JOIN group_members gm ON g.id = gm.group_id 
                 WHERE gm.user_id = $1`,
                [userId]
            );
            cb({ status: "ok", groups: res.rows });
        } catch (error) {
            console.error("Error listing groups:", error);
            cb({ status: "error", message: "Failed to list groups" });
        }
    });

    socket.on("group:get", async (payload: { groupId: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Verify membership
            const memberCheck = await pool.query(
                "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2",
                [payload.groupId, userId]
            );

            if (memberCheck.rows.length === 0) {
                return cb({ status: "error", message: "Not a member of this group" });
            }

            const groupRes = await pool.query("SELECT * FROM groups WHERE id = $1", [payload.groupId]);
            const membersRes = await pool.query(
                `SELECT u.id, u.username, u.email, gm.role FROM users u 
                 JOIN group_members gm ON u.id = gm.user_id 
                 WHERE gm.group_id = $1`,
                [payload.groupId]
            );

            cb({
                status: "ok",
                group: groupRes.rows[0],
                members: membersRes.rows
            });
        } catch (error) {
            console.error("Error getting group details:", error);
            cb({ status: "error", message: "Failed to get group details" });
        }
    });

    socket.on("group:add_member", async (payload: { groupId: string, email: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Check if requester is admin
            const adminCheck = await pool.query(
                "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2 AND role = 'admin'",
                [payload.groupId, userId]
            );

            if (adminCheck.rows.length === 0) {
                return cb({ status: "error", message: "Only admins can add members" });
            }

            // Find user by email
            const userRes = await pool.query("SELECT id FROM users WHERE email = $1", [payload.email]);
            if (userRes.rows.length === 0) {
                return cb({ status: "error", message: "User not found" });
            }
            const newMemberId = userRes.rows[0].id;

            // Check if already a member
            const existingMember = await pool.query(
                "SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2",
                [payload.groupId, newMemberId]
            );
            if (existingMember.rows.length > 0) {
                return cb({ status: "error", message: "User is already a member" });
            }

            await pool.query(
                "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)",
                [payload.groupId, newMemberId, 'member']
            );

            cb({ status: "ok" });
        } catch (error) {
            console.error("Error adding group member:", error);
            cb({ status: "error", message: "Failed to add member" });
        }
    });
};
