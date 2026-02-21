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

export const adminOnly = async (req: Request, res: Response, next: NextFunction) => {
    if (!res.locals.session || !res.locals.user) {
        return res.status(401).json({ error: "Unauthorized" });
    }

    try {
        // Evaluate user_roles -> system_roles
        const adminCheck = await pool.query(
            `SELECT sr.name 
             FROM user_roles ur
             JOIN system_roles sr ON ur.role_id = sr.id
             WHERE ur.user_id = $1`,
            [res.locals.user.id]
        );

        if (adminCheck.rows.length === 0) {
            return res.status(403).json({ error: "Forbidden: Admin access required." });
        }

        // Optional: Attach roles to locals
        res.locals.system_roles = adminCheck.rows.map(r => r.name);
        
        next();
    } catch (error) {
        console.error("Admin verification error", error);
        return res.status(500).json({ error: "Internal server error" });
    }
};
