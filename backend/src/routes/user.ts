import { Router } from "express";
import { pool } from "../db";
import { requireAuth, verifySession } from "../middleware/auth";

const router = Router();

// Apply session verification
router.use(verifySession);

router.get("/profile", requireAuth, async (req, res) => {
    const userId = res.locals.user!.id;

    try {
        const query = `
            SELECT 
                u.id, u.username, u.email, u.avatar_url,
                p.first_name, p.last_name, p.birthday, p.mobile_phone as phone
            FROM users u
            LEFT JOIN user_profiles p ON u.id = p.user_id
            WHERE u.id = $1
        `;
        const result = await pool.query(query, [userId]);
        const user = result.rows[0];

        if (!user) {
            return res.status(404).json({ error: "User not found" });
        }

        const profile = {
            ...user,
            real_name: user.first_name && user.last_name ? `${user.first_name} ${user.last_name}` : (user.first_name || user.last_name || "")
        };

        return res.json({ status: "ok", profile });
    } catch (error) {
        console.error("Error fetching profile:", error);
        return res.status(500).json({ error: "Failed to fetch profile" });
    }
});

router.patch("/profile", requireAuth, async (req, res) => {
    const userId = res.locals.user!.id;
    const { real_name, birthday, phone, avatar_url } = req.body;

    try {
        await pool.query("BEGIN");

        if (avatar_url !== undefined) {
            await pool.query("UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2", [avatar_url, userId]);
        }

        let firstName = "";
        let lastName = "";
        if (real_name) {
            const parts = real_name.trim().split(/\s+/);
            firstName = parts[0];
            lastName = parts.slice(1).join(" ");
        }

        const upsertProfileQuery = `
            INSERT INTO user_profiles (user_id, first_name, last_name, birthday, mobile_phone, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                first_name = COALESCE(NULLIF($2, ''), user_profiles.first_name),
                last_name = COALESCE(NULLIF($3, ''), user_profiles.last_name),
                birthday = COALESCE($4, user_profiles.birthday),
                mobile_phone = COALESCE(NULLIF($5, ''), user_profiles.mobile_phone),
                updated_at = NOW()
        `;
        await pool.query(upsertProfileQuery, [userId, firstName, lastName, birthday || null, phone || null]);

        await pool.query("COMMIT");
        return res.json({ status: "ok", message: "Profile updated" });
    } catch (error) {
        await pool.query("ROLLBACK");
        console.error("Error updating profile:", error);
        return res.status(500).json({ error: "Failed to update profile" });
    }
});

export default router;
