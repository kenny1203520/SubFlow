import { Server, Socket } from "socket.io";
import { pool } from "../db";

export const registerGroupHandlers = (io: Server, socket: Socket) => {
    socket.on("group:create", async (payload: {
        name: string,
        service_name?: string,
        website?: string,
        plan_name?: string,
        amount?: number,
        currency?: string,
        billing_cycle?: string,
        max_members?: number,
        billing_method?: 'equal' | 'fixed' | 'percentage',
        initial_members?: { email?: string, name?: string }[],
        next_payment_date?: string
    }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Icon handling
            let iconUrl = '';
            if (payload.website) {
                // Determine domain for icon
                try {
                    const domain = new URL(payload.website).hostname;
                    iconUrl = `https://www.google.com/s2/favicons?domain=${domain}&sz=64`;
                } catch (e) {
                    // Invalid URL, ignore
                }
            }

            const client = await pool.connect();
            try {
                await client.query("BEGIN");

                let serviceId = null;
                if (payload.service_name) {
                    // Check if service exists
                    const serviceRes = await client.query("SELECT id FROM services WHERE name = $1", [payload.service_name]);
                    if (serviceRes.rows.length > 0) {
                        serviceId = serviceRes.rows[0].id;
                    } else {
                        // Create new service
                        const newService = await client.query(
                            "INSERT INTO services (name, domain, icon_url, created_by) VALUES ($1, $2, $3, $4) RETURNING id",
                            [payload.service_name, payload.website || '', iconUrl, userId]
                        );
                        serviceId = newService.rows[0].id;
                    }
                }

                const groupRes = await client.query(
                    `INSERT INTO groups (
                        name, service_name, service_id, website, plan_name, amount, 
                        currency, billing_cycle, max_members, billing_method, created_by,
                        next_payment_date
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING *`,
                    [
                        payload.name, payload.service_name, serviceId, payload.website, payload.plan_name,
                        payload.amount, payload.currency || 'TWD', payload.billing_cycle,
                        payload.max_members, payload.billing_method || 'equal', userId,
                        payload.next_payment_date ? new Date(payload.next_payment_date) : null
                    ]
                );
                const group = groupRes.rows[0];

                // Add creator as admin
                await client.query(
                    "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)",
                    [group.id, userId, 'admin']
                );

                // Add initial members if any
                if (payload.initial_members && payload.initial_members.length > 0) {
                    for (const member of payload.initial_members) {
                        if (member.email) {
                            // Find registered user
                            const userRes = await client.query("SELECT id FROM users WHERE email = $1", [member.email]);
                            if (userRes.rows.length > 0) {
                                await client.query(
                                    "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
                                    [group.id, userRes.rows[0].id, 'member']
                                );
                            } else {
                                // Add as non-member with email as temp_name or handle as приглашение
                                await client.query(
                                    "INSERT INTO group_members (group_id, temp_name, role) VALUES ($1, $2, $3)",
                                    [group.id, member.email, 'member']
                                );
                            }
                        } else if (member.name) {
                            await client.query(
                                "INSERT INTO group_members (group_id, temp_name, role) VALUES ($1, $2, $3)",
                                [group.id, member.name, 'member']
                            );
                        }
                    }
                }

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
                `SELECT gm.id as member_id, u.id as user_id, COALESCE(u.username, gm.temp_name) as username, 
                        u.email, gm.role, gm.temp_name 
                 FROM group_members gm 
                 LEFT JOIN users u ON gm.user_id = u.id 
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

    socket.on("group:add_member", async (payload: { groupId: string, email?: string, name?: string }, cb: (res: any) => void) => {
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

            if (payload.email) {
                // Find user by email
                const userRes = await pool.query("SELECT id FROM users WHERE email = $1", [payload.email]);
                if (userRes.rows.length === 0) {
                    // Add as non-member with email as temp_name
                    await pool.query(
                        "INSERT INTO group_members (group_id, temp_name, role) VALUES ($1, $2, $3)",
                        [payload.groupId, payload.email, 'member']
                    );
                    return cb({ status: "ok", message: "User added as non-member participant" });
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
            } else if (payload.name) {
                await pool.query(
                    "INSERT INTO group_members (group_id, temp_name, role) VALUES ($1, $2, $3)",
                    [payload.groupId, payload.name, 'member']
                );
            } else {
                return cb({ status: "error", message: "Email or Name is required" });
            }

            cb({ status: "ok" });
        } catch (error) {
            console.error("Error adding group member:", error);
            cb({ status: "error", message: "Failed to add member" });
        }
    });

    socket.on("group:bind_member", async (payload: { memberId: string }, cb: (res: any) => void) => {
        try {
            const userId = socket.data.user.id;

            // Check if member exists and has no user_id
            const memberRes = await pool.query(
                "SELECT group_id, user_id FROM group_members WHERE id = $1",
                [payload.memberId]
            );

            if (memberRes.rows.length === 0) {
                return cb({ status: "error", message: "Member record not found" });
            }

            if (memberRes.rows[0].user_id) {
                return cb({ status: "error", message: "Member is already bound to a user" });
            }

            // Bind current user to this member slot
            await pool.query(
                "UPDATE group_members SET user_id = $1, temp_name = NULL WHERE id = $2",
                [userId, payload.memberId]
            );

            cb({ status: "ok" });
        } catch (error) {
            console.error("Error binding member:", error);
            cb({ status: "error", message: "Failed to bind member" });
        }
    });
};
