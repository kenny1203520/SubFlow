import { pool } from "../db";
import crypto from "crypto";

// Helper to get device fingerprint
export const getDeviceFingerprint = (req: any) => {
    // In a real app, use a proper library or client-sent fingerprint
    const ua = req.headers?.['user-agent'] || req.handshake?.headers?.['user-agent'] || '';
    const ip = req.ip || req.handshake?.address || '';
    return crypto.createHash('sha256').update(`${ua}|${ip}`).digest('hex');
};

export const logActivity = async (userId: string | null, behaviorType: string, action: string, risk: 'info' | 'low' | 'medium' | 'high' | 'critical', description: string, req: any, deviceFingerprint?: string) => {
    try {
        const fp = deviceFingerprint || getDeviceFingerprint(req);
        // Handle socket request objects which might be different structure
        const ip = req.ip || req.handshake?.address || '';
        const userAgent = req.headers?.['user-agent'] || req.handshake?.headers?.['user-agent'] || '';

        await pool.query(
            `INSERT INTO activity_logs (user_id, action, behavior_type, risk_level, description, ip_address, user_agent, device_fingerprint)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
            [userId, action, behaviorType, risk, description, ip, userAgent, fp]
        );
    } catch (e) {
        console.error("Failed to log activity:", e);
    }
};
