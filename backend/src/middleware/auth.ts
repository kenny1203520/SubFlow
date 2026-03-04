import { Request, Response, NextFunction } from "express";
import { lucia } from "../auth/lucia";
import { runWithContext } from "../utils/context";
import { pool } from "../db";

export const verifySession = async (req: Request, res: Response, next: NextFunction) => {
    const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "");
    if (!sessionId) {
        res.locals.user = null;
        res.locals.session = null;
        return runWithContext({ userId: null }, next);
    }

    const { session, user } = await lucia.validateSession(sessionId);
    if (session && session.fresh) {
        res.setHeader("Set-Cookie", lucia.createSessionCookie(session.id).serialize());
    }
    if (!session) {
        res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
        res.locals.user = null;
        res.locals.session = null;
        return runWithContext({ userId: null }, next);
    }

    // Check suspension/ban status
    const securityCheck = await pool.query(
        "SELECT is_suspended, suspended_until, is_blocked FROM user_security WHERE user_id = $1",
        [user.id]
    );
    
    if (securityCheck.rows.length > 0) {
        const security = securityCheck.rows[0];
        if (security.is_blocked) {
            await lucia.invalidateSession(session.id);
            res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
            res.locals.user = null;
            res.locals.session = null;
            return res.status(403).json({ error: "Account is blocked." });
        }
        if (security.is_suspended) {
            if (security.suspended_until && new Date(security.suspended_until) <= new Date()) {
                // Suspension expired, clear it in DB
                await pool.query(
                    "UPDATE user_security SET is_suspended = FALSE, suspended_until = NULL, failed_login_attempts = 0 WHERE user_id = $1",
                    [user.id]
                );
            } else {
                await lucia.invalidateSession(session.id);
                res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
                res.locals.user = null;
                res.locals.session = null;
                return res.status(403).json({ error: "Account is temporarily suspended." });
            }
        }
    }

    res.locals.user = user;
    res.locals.session = session;

    return runWithContext({ userId: user?.id ?? null }, next);
};

export const requireAuth = (req: Request, res: Response, next: NextFunction) => {
    if (!res.locals.session) {
        return res.status(401).json({ error: "Unauthorized" });
    }
    next();
};

/**
 * Middleware to attach user to req object for easier access
 * Works with existing verifySession middleware
 */
export const authenticateSession = (req: Request, res: Response, next: NextFunction) => {
    if (!res.locals.session || !res.locals.user) {
        return res.status(401).json({ error: "Unauthorized" });
    }
    // Attach user to request for easier access in controllers
    (req as any).user = res.locals.user;
    (req as any).session = res.locals.session;
    next();
};

export const requirePermission = (scope: string, action: string, resource: string) => {
    return async (req: Request, res: Response, next: NextFunction) => {
        if (!res.locals.session || !res.locals.user) {
            return res.status(401).json({ error: "Unauthorized" });
        }

        try {
            const userId = res.locals.user.id;

            // 1. Check if user has Administrator role (always allowed)
            const rolesResult = await pool.query(
                `SELECT sr.name FROM user_roles ur 
                 JOIN system_roles sr ON ur.role_id = sr.id 
                 WHERE ur.user_id = $1`,
                [userId]
            );
            const roles = rolesResult.rows.map(r => r.name);
            if (roles.includes('Administrator')) {
                next();
                return;
            }

            // 2. Check direct permissions
            const directPermCheck = await pool.query(
                `SELECT 1 FROM permissions_user up 
                 JOIN permissions p ON up.permission_id = p.id 
                 WHERE up.user_id = $1 AND p.scope = $2 AND p.action = $3 AND p.resource = $4`,
                [userId, scope, action, resource]
            );

            if (directPermCheck.rowCount && directPermCheck.rowCount > 0) {
                next();
                return;
            }

            // 3. Check role-based permissions
            const rolePermCheck = await pool.query(
                `SELECT 1 FROM user_roles ur 
                 JOIN permissions_system_role psr ON ur.role_id = psr.role_id 
                 JOIN permissions p ON psr.permission_id = p.id 
                 WHERE ur.user_id = $1 AND p.scope = $2 AND p.action = $3 AND p.resource = $4`,
                [userId, scope, action, resource]
            );

            if (rolePermCheck.rowCount && rolePermCheck.rowCount > 0) {
                next();
                return;
            }

            return res.status(403).json({ error: `Forbidden: Missing required permission ${scope}:${action}:${resource}` });
        } catch (error) {
            console.error("Permission verification error", error);
            return res.status(500).json({ error: "Internal server error" });
        }
    };
};
