import express from "express";
import { Router } from "express";
import { pool } from "../db";
import { requireAuth } from "../middleware/auth";
import { generateId } from "lucia";

const router = Router();

router.use(requireAuth);

interface SubSplit {
    userId: string;
    sharePercentage: number; // e.g., 50.00
}

// Create a subscription
router.post("/", createSubscriptionHandler);

// List user's subscriptions
router.get("/", listSubscriptionsHandler);

async function createSubscriptionHandler(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    const { serviceName, amount, cycle, nextPaymentDate, splits } = req.body;

    if (!serviceName || !amount || !cycle || !['monthly', 'yearly'].includes(cycle)) {
        return res.status(400).send("Invalid input");
    }

    const client = await pool.connect();
    try {
        await client.query("BEGIN");

        const subId = generateId(15);
        await client.query(
            "INSERT INTO subscriptions (id, owner_id, service_name, amount, cycle, next_payment_date) VALUES ($1, $2, $3, $4, $5, $6)",
            [subId, userId, serviceName, amount, cycle, nextPaymentDate || new Date()]
        );

        if (splits && Array.isArray(splits)) {
            for (const split of splits as SubSplit[]) {
                await client.query(
                    "INSERT INTO subscription_splits (subscription_id, user_id, share_percentage) VALUES ($1, $2, $3)",
                    [subId, split.userId, split.sharePercentage]
                );
            }
        }

        await client.query("COMMIT");
        res.status(201).json({ id: subId, message: "Subscription created" });
    } catch (error) {
        await client.query("ROLLBACK");
        console.error(error);
        res.status(500).send("Server Error");
    } finally {
        client.release();
    }
}

async function listSubscriptionsHandler(req: express.Request, res: express.Response) {
    const userId = res.locals.user.id;
    try {
        const result = await pool.query(`
            SELECT s.*, 
            (SELECT json_agg(json_build_object('user_id', ss.user_id, 'share_percentage', ss.share_percentage, 'username', u.username))
             FROM subscription_splits ss
             JOIN users u ON ss.user_id = u.id
             WHERE ss.subscription_id = s.id
            ) as splits
            FROM subscriptions s
            WHERE s.owner_id = $1
            ORDER BY s.next_payment_date ASC
        `, [userId]);
        res.json(result.rows);
    } catch (error) {
        console.error(error);
        res.status(500).send("Server Error");
    }
}

export default router;
