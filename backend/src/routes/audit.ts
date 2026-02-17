import express from 'express';
import { pool } from '../db';
import { requireAuth } from '../middleware/auth';

const router = express.Router();

router.use(requireAuth);

// Get User Activity Logs
router.get('/user/activity', async (req, res) => {
    const userId = res.locals.user.id;
    const limit = parseInt(req.query.limit as string) || 20;
    const offset = parseInt(req.query.offset as string) || 0;

    try {
        const query = `
            SELECT 
                id, 
                action, 
                behavior_type, 
                description, 
                created_at, 
                ip_address,
                risk_level
            FROM activity_logs
            WHERE user_id = $1
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
        `;
        const result = await pool.query(query, [userId, limit, offset]);

        // Get total count for pagination
        const countQuery = `SELECT count(*) FROM activity_logs WHERE user_id = $1`;
        const countResult = await pool.query(countQuery, [userId]);
        const total = parseInt(countResult.rows[0].count);

        res.json({
            logs: result.rows,
            pagination: {
                total,
                limit,
                offset
            }
        });
    } catch (error) {
        console.error('Error fetching activity logs:', error);
        console.error('User ID:', userId);
        console.error('Query Params:', { limit, offset });
        res.status(500).json({ message: 'Server error fetching activity logs', error: String(error) });
    }
});

export default router;
