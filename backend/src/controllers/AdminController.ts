import { Request, Response } from 'express';
import { pool } from '../db';
import { SystemSettingsService } from '../services/SystemSettingsService';
import { Scrypt } from 'oslo/password';
import crypto from 'crypto';

const scrypt = new Scrypt();
const getPepperedPassword = (password: string) => {
    const authSecret = process.env.AUTH_SECRET;
    if (!authSecret) {
        throw new Error("AUTH_SECRET is not defined in environment variables");
    }
    return crypto.createHmac('sha256', authSecret).update(password).digest('hex');
};

export class AdminController {
    
    // --- Dashboard Stats ---
    static async getStats(req: Request, res: Response) {
        try {
            const usersCount = await pool.query('SELECT COUNT(*) FROM users');
            const blockedCount = await pool.query('SELECT COUNT(*) FROM user_security WHERE is_blocked = true');
            const suspendedCount = await pool.query('SELECT COUNT(*) FROM user_security WHERE is_suspended = true');
            const sessionsCount = await pool.query('SELECT COUNT(*) FROM sessions WHERE expires_at > NOW()');
            const ipBlocksCount = await pool.query('SELECT COUNT(*) FROM ip_blocks WHERE (expires_at IS NULL OR expires_at > NOW())');

            res.json({
                status: 'ok',
                stats: {
                    totalUsers: parseInt(usersCount.rows[0].count),
                    blockedUsers: parseInt(blockedCount.rows[0].count),
                    suspendedUsers: parseInt(suspendedCount.rows[0].count),
                    activeSessions: parseInt(sessionsCount.rows[0].count),
                    blockedIps: parseInt(ipBlocksCount.rows[0].count)
                }
            });
        } catch (error) {
            console.error('Failed to get stats:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

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
        const { action, reason, until } = req.body;

        try {
            if (action === 'ban') {
                await pool.query(
                    `UPDATE user_security SET is_blocked = true, block_reason = $1, blocked_at = NOW() WHERE user_id = $2`,
                    [reason, id]
                );
            } else if (action === 'unban') {
                await pool.query(
                    `UPDATE user_security SET is_blocked = false, block_reason = null, blocked_at = null WHERE user_id = $1`,
                    [id]
                );
            } else if (action === 'suspend') {
                const suspendUntil = until || new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
                await pool.query(
                    `UPDATE user_security SET is_suspended = true, suspended_until = $1, suspension_reason = $2, suspended_at = NOW() WHERE user_id = $3`,
                    [suspendUntil, reason, id]
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
            
            const settings = result.rows.reduce((acc: Record<string, any>, row: any) => {
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
            await SystemSettingsService.setSetting(key, value, userId);
            res.json({ status: 'ok', message: 'Setting updated successfully' });
        } catch (error) {
            console.error('Failed to update setting:', error);
            res.status(500).json({ status: 'error', message: 'Internal server error' });
        }
    }

    // --- Role and Permission Management ---

    static async getSystemRoles(req: Request, res: Response) {
        try {
            const roles = await pool.query('SELECT * FROM system_roles ORDER BY name');
            res.json({ status: 'ok', roles: roles.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async getUserRoles(req: Request, res: Response) {
        const { id } = req.params;
        try {
            const roles = await pool.query(
                `SELECT sr.* FROM user_roles ur JOIN system_roles sr ON ur.role_id = sr.id WHERE ur.user_id = $1`,
                [id]
            );
            res.json({ status: 'ok', roles: roles.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async assignUserRole(req: Request, res: Response) {
        const { id } = req.params;
        const { roleId } = req.body;
        try {
            await pool.query(
                `INSERT INTO user_roles (user_id, role_id, assigned_by) VALUES ($1, $2, $3) ON CONFLICT (user_id, role_id) DO NOTHING`,
                [id, roleId, res.locals.user?.id]
            );
            res.json({ status: 'ok' });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async removeUserRole(req: Request, res: Response) {
        const { id, roleId } = req.params;
        try {
            await pool.query(`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, [id, roleId]);
            res.json({ status: 'ok' });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    // --- IP Blocks ---

    static async getIpBlocks(req: Request, res: Response) {
        try {
            const blocks = await pool.query(
                `SELECT ib.*, u.username as created_by_name 
                 FROM ip_blocks ib 
                 LEFT JOIN users u ON ib.created_by = u.id 
                 ORDER BY ib.created_at DESC`
            );
            res.json({ status: 'ok', blocks: blocks.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async blockIp(req: Request, res: Response) {
        const { ip, reason, expiresAt } = req.body;
        const userId = res.locals.user?.id;
        try {
            await pool.query(
                `INSERT INTO ip_blocks (ip_address, reason, expires_at, created_by)
                 VALUES ($1::inet, $2, $3, $4)
                 ON CONFLICT (ip_address) DO UPDATE SET reason = $2, expires_at = $3, created_by = $4`,
                [ip, reason || null, expiresAt || null, userId]
            );
            res.json({ status: 'ok' });
        } catch (error) {
            console.error('Failed to block IP:', error);
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async unblockIp(req: Request, res: Response) {
        const { ip } = req.params;
        try {
            await pool.query('DELETE FROM ip_blocks WHERE ip_address = $1::inet', [ip]);
            res.json({ status: 'ok' });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    // --- Admin Password Reset ---
    static async changeUserPassword(req: Request, res: Response) {
        const { id } = req.params;
        const { newPassword } = req.body;
        if (!newPassword || newPassword.length < 6) {
            return res.status(400).json({ status: 'error', message: 'Password must be at least 6 characters' });
        }
        try {
            const passwordHash = await scrypt.hash(getPepperedPassword(newPassword));
            await pool.query('UPDATE users SET password_hash = $1 WHERE id = $2', [passwordHash, id]);
            res.json({ status: 'ok', message: 'Password changed successfully' });
        } catch (error) {
            console.error('Failed to change password:', error);
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    // --- Permission Management ---
    static async getPermissions(req: Request, res: Response) {
        try {
            const result = await pool.query('SELECT * FROM permissions ORDER BY scope, action, resource');
            res.json({ status: 'ok', permissions: result.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async getRolePermissions(req: Request, res: Response) {
        const { roleId } = req.params;
        try {
            const result = await pool.query(
                `SELECT p.* FROM permissions_system_role psr 
                 JOIN permissions p ON psr.permission_id = p.id 
                 WHERE psr.role_id = $1 ORDER BY p.scope, p.action`,
                [roleId]
            );
            res.json({ status: 'ok', permissions: result.rows });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async addRolePermission(req: Request, res: Response) {
        const { roleId } = req.params;
        const { permissionId } = req.body;
        try {
            await pool.query(
                `INSERT INTO permissions_system_role (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
                [roleId, permissionId]
            );
            res.json({ status: 'ok' });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }

    static async removeRolePermission(req: Request, res: Response) {
        const { roleId, permissionId } = req.params;
        try {
            await pool.query(
                `DELETE FROM permissions_system_role WHERE role_id = $1 AND permission_id = $2`,
                [roleId, permissionId]
            );
            res.json({ status: 'ok' });
        } catch (error) {
            res.status(500).json({ status: 'error', message: 'Internal Server Error' });
        }
    }
}
