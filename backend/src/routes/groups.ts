import { Router } from "express";
import { pool } from "../db";
import { requireAuth } from "../middleware/auth";
import { generateId } from "lucia";

const router = Router();

router.use(requireAuth);

// List user's groups
router.get("/", async (req, res) => {
    const userId = res.locals.user.id;
    try {
        const result = await pool.query(`
            SELECT g.*, gm.role 
            FROM groups g
            JOIN group_members gm ON g.id = gm.group_id
            WHERE gm.user_id = $1
            ORDER BY g.created_at DESC
        `, [userId]);
        res.json(result.rows);
    } catch (error) {
        console.error(error);
        res.status(500).send("Server Error");
    }
});

// Create a group
router.post("/", async (req, res) => {
    const userId = res.locals.user.id;
    const { name } = req.body;

    if (typeof name !== "string" || name.trim().length === 0) {
        return res.status(400).send("Invalid group name");
    }

    const client = await pool.connect();
    try {
        await client.query("BEGIN");
        const groupId = generateId(15); // Or depend on DB default if using gen_random_uuid in insert, but explicit is fine or returning

        // We can just use DEFAULT for id if we want, but returning it is easier
        const groupResult = await client.query(
            "INSERT INTO groups (name, created_by) VALUES ($1, $2) RETURNING id, name, created_at",
            [name, userId]
        );
        const newGroup = groupResult.rows[0];

        await client.query(
            "INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, 'admin')",
            [newGroup.id, userId]
        );

        await client.query("COMMIT");
        res.status(201).json(newGroup);
    } catch (error) {
        await client.query("ROLLBACK");
        console.error(error);
        res.status(500).send("Server Error");
    } finally {
        client.release();
    }
});

// Get group details
router.get("/:id", async (req, res) => {
    const userId = res.locals.user.id;
    const groupId = req.params.id;

    try {
        // Check membership
        const memberCheck = await pool.query(
            "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        if (memberCheck.rows.length === 0) {
            return res.status(403).send("Not a member of this group");
        }

        const groupResult = await pool.query("SELECT * FROM groups WHERE id = $1", [groupId]);
        const membersResult = await pool.query(`
            SELECT u.id, u.username, u.email, gm.role
            FROM group_members gm
            JOIN users u ON gm.user_id = u.id
            WHERE gm.group_id = $1
        `, [groupId]);

        // Get recent expenses? Maybe unrelated to this route, but useful.
        // Let's keep it simple: Group info + Members

        res.json({
            ...groupResult.rows[0],
            members: membersResult.rows
        });
    } catch (error) {
        console.error(error);
        res.status(500).send("Server Error");
    }
});

export default router;
