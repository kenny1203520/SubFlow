import express from "express";
import { Router } from "express";
import { pool } from "../db";
import { requireAuth, requirePermission } from "../middleware/auth";

const router = Router();

router.use(requireAuth);

// Get user profile
router.get("/profile", requirePermission('user', 'read', 'profile'), getProfile);

// Get user profile by ID
router.get("/profile/:id", requirePermission('users', 'read', 'profile'), getProfileById);

// Update user profile
router.patch("/profile", requirePermission('user', 'update', 'profile'), updateProfile);

async function getProfile(req: express.Request, res: express.Response) {
    const userId = res.locals.user!.id;

    try {
        const query = `
            SELECT 
                u.id, u.username, u.email, u.avatar_url,
                p.first_name, p.middle_name, p.last_name, p.nickname,
                p.birthday, p.mobile_phone as phone,
                p.id_number, p.passport_number, p.address
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
            // Fallback for real_name display if needed, but frontend should use individual fields
            real_name: [user.first_name, user.middle_name, user.last_name].filter(Boolean).join(' ')
        };

        return res.json({ status: "ok", profile });
    } catch (error) {
        console.error("Error fetching profile:", error);
        return res.status(500).json({ error: "Failed to fetch profile" });
    }
}

async function getProfileById(req: express.Request, res: express.Response) {
    const userId = req.params.id;

    try {
        const query = `
            SELECT 
                u.id, u.username, u.email, u.avatar_url,
                p.first_name, p.middle_name, p.last_name, p.nickname,
                p.birthday, p.mobile_phone as phone,
                p.id_number, p.passport_number, p.address
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
            // Fallback for real_name display if needed, but frontend should use individual fields
            real_name: [user.first_name, user.middle_name, user.last_name].filter(Boolean).join(' ')
        };

        return res.json({ status: "ok", profile });
    } catch (error) {
        console.error("Error fetching profile:", error);
        return res.status(500).json({ error: "Failed to fetch profile" });
    }
}

async function updateProfile(req: express.Request, res: express.Response) {
    const userId = res.locals.user!.id;
    const {
        first_name, middle_name, last_name, nickname,
        birthday, phone, avatar_url,
        id_number, passport_number, address
    } = req.body;

    try {
        await pool.query("BEGIN");

        if (avatar_url !== undefined) {
            await pool.query("UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2", [avatar_url, userId]);
        }

        const upsertProfileQuery = `
            INSERT INTO user_profiles (
                user_id, first_name, middle_name, last_name, nickname,
                birthday, mobile_phone, id_number, passport_number, address, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                first_name = COALESCE($2, user_profiles.first_name),
                middle_name = COALESCE($3, user_profiles.middle_name),
                last_name = COALESCE($4, user_profiles.last_name),
                nickname = COALESCE($5, user_profiles.nickname),
                birthday = COALESCE($6, user_profiles.birthday),
                mobile_phone = COALESCE($7, user_profiles.mobile_phone),
                id_number = COALESCE($8, user_profiles.id_number),
                passport_number = COALESCE($9, user_profiles.passport_number),
                address = COALESCE($10, user_profiles.address),
                updated_at = NOW()
        `;

        await pool.query(upsertProfileQuery, [
            userId,
            first_name || null,
            middle_name || null,
            last_name || null,
            nickname || null,
            birthday || null,
            phone || null,
            id_number || null,
            passport_number || null,
            address || null
        ]);

        await pool.query("COMMIT");
        return res.json({ status: "ok", message: "Profile updated" });
    } catch (error) {
        await pool.query("ROLLBACK");
        console.error("Error updating profile:", error);
        return res.status(500).json({ error: "Failed to update profile" });
    }
}

export default router;
