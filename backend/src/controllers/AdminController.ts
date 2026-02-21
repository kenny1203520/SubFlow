import { Request, Response } from 'express';
import { pool } from '../db';
import { SystemSettingService } from '../services/SystemSettingService';

export class AdminController {
    
    // --- User Management ---

    static async getUsers(req: Request, res: Response) {
        try {
            const result = await pool.query(`
                SELECT u.id, u.username, u.email, u.created_at, 
                       s.is_suspended, s.suspended_until, s.is_blocked, s.two_factor_enabled
                FROM users u
                LEFT JOIN user_security s ON u.id = s.user_id
                ORDER BY u.created_at DESC
            `);
            res.json({ status: 'ok', users: result.rows });
        } catch (error) {
            console.error('Failed to get users:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    static async updateUserStatus(req: Request, res: Response) {
        const { id } = req.params;
        const { action, reason, until } = req.body; // action: 'ban', 'unban', 'suspend', 'unsuspend'

        try {
            if (action === 'ban') {
                await pool.query(
                    `UPDATE user_security SET is_blocked = true, is_suspended = false, suspended_until = null, block_reason = $1, blocked_at = NOW() WHERE user_id = $2`,
                    [reason, id]
                );
            } else if (action === 'unban') {
                await pool.query(
                    `UPDATE user_security SET is_blocked = false, block_reason = null, blocked_at = null WHERE user_id = $1`,
                    [id]
                );
            } else if (action === 'suspend') {
                await pool.query(
                    `UPDATE user_security SET is_suspended = true, suspended_until = $1, suspension_reason = $2, suspended_at = NOW() WHERE user_id = $3`,
                    [until, reason, id]
                );
            } else if (action === 'unsuspend') {
                await pool.query(
                    `UPDATE user_security SET is_suspended = false, suspended_until = null, suspension_reason = null, suspended_at = null WHERE user_id = $1`,
                    [id]
                );
            } else {
                return res.status(400).json({ status: 'error', message: 'Invalid action' });
            }

            res.json({ status: 'ok', message: 'User status updated successfully' });
        } catch (error) {
            console.error('Failed to update user status:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    static async getUserSessions(req: Request, res: Response) {
        const { id } = req.params;
        try {
            const sessions = await pool.query(
                `SELECT id, ip_address, user_agent, device_fingerprint, created_at, expires_at 
                 FROM sessions WHERE user_id = $1 ORDER BY created_at DESC`,
                [id]
            );
            res.json({ status: 'ok', sessions: sessions.rows });
        } catch (error) {
            console.error('Failed to get user sessions:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    static async revokeUserSession(req: Request, res: Response) {
        const { id, sessionId } = req.params;
        try {
            // Only allow deleting sessions belonging to the specified user
            const result = await pool.query(
                `DELETE FROM sessions WHERE id = $1 AND user_id = $2 RETURNING id`,
                [sessionId, id]
            );
            
            if (result.rowCount === 0) {
                return res.status(404).json({ status: 'error', message: 'Session not found or already revoked' });
            }

            res.json({ status: 'ok', message: 'Session revoked successfully' });
        } catch (error) {
            console.error('Failed to revoke session:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }


    // --- System Settings Management ---

    static async getSettings(req: Request, res: Response) {
        try {
            const result = await pool.query('SELECT key, value, description, updated_at FROM system_settings');
            
            // Map rows to a more JS-friendly object
            const settings = result.rows.reduce((acc, row) => {
                acc[row.key] = row.value;
                return acc;
            }, {} as Record<string, any>);
            
            res.json({ status: 'ok', settings, list: result.rows });
        } catch (error) {
            console.error('Failed to get settings:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    static async updateSetting(req: Request, res: Response) {
        const { key, value } = req.body;
        const userId = res.locals.user?.id;

        if (!key || value === undefined) {
            return res.status(400).json({ status: 'error', message: 'Key and value are required' });
        }

        try {
            const success = await SystemSettingService.setSetting(key, value, userId);
            if (success) {
                res.json({ status: 'ok', message: 'Setting updated successfully' });
            } else {
                res.status(500).json({ status: 'error', message: 'Failed to update setting' });
            }
        } catch (error) {
            console.error('Failed to update setting:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    // --- Role and Permission Management (Optional basic list for global roles) ---

    static async getSystemRoles(req: Request, res: Response) {
        try {
            const roles = await pool.query('SELECT * FROM system_roles ORDER BY name');
            res.json({ status: 'ok', roles: roles.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }
}
