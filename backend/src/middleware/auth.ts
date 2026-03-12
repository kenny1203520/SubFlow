import { Request, Response, NextFunction } from "express";
import { lucia } from "../auth/lucia";
import { runWithContext } from "../utils/context";
import { pool } from "../db";
import { logActivity, getDeviceFingerprint } from '../utils/audit';
import { SystemSettingService } from "../services/SystemSettingService";
import { AuthRepository } from "../repositories/AuthRepository";
import { RBACService } from "../services/RBACService";

/**
 * Enhanced authentication and security middleware for Express routes, including:
 * - IP blocking
 * - CAPTCHA verification
 */
class AuthMiddleware {
    private systemSettingService = new SystemSettingService();
    private authRepo = new AuthRepository();
    private rbacService = new RBACService();

    /**
     * Check if the incoming request's IP address is blocked.
     * If blocked, log the attempt and return 403.
     * @param req 
     * @param res 
     * @param next 
     * @returns 
     */
    checkIpBlocked = async (req: Request, res: Response, next: NextFunction) => {
        const ipAddress = req.ip || '';
        
        try {
            const isBlocked = await this.authRepo.isIpBlocked(ipAddress);
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked request from IP: ${ipAddress}`,
                    req, 
                    getDeviceFingerprint(req)
                );
                return res.status(403).json({ 
                    error: "auth.errors.accessDenied"
                });
            }
            next();
        } catch (error) {
            console.error("IP block check error:", error);
            next(); // Fail open
        }
    };

    /**
     * Verify CAPTCHA token from the request body based on the configured provider.
     * @param req 
     * @param res 
     * @param next 
     * @returns 
     */
    verifyCaptcha = async (req: Request, res: Response, next: NextFunction) => {
        try {
            const captcha = await this.systemSettingService.getCaptchaSettings();
            
            if (captcha.provider === "none") {
                return next();
            }

            const token = req.body.captchaToken;
            if (!token) {
                return res.status(400).json({ 
                    error: "auth.errors.captchaRequired" 
                });
            }

            let verifyUrl = "";
            if (captcha.provider === "turnstile") {
                verifyUrl = "https://challenges.cloudflare.com/turnstile/v0/siteverify";
            } else if (captcha.provider === "recaptcha") {
                verifyUrl = "https://www.google.com/recaptcha/api/siteverify";
            } else {
                return res.status(500).json({ 
                    error: "auth.errors.invalidCaptchaProvider" 
                });
            }

            const response = await fetch(verifyUrl, {
                method: "POST",
                headers: { "Content-Type": "application/x-www-form-urlencoded" },
                body: `secret=${captcha.secretKey}&response=${token}&remoteip=${req.ip}`
            });

            const data = await response.json();
            if (!data.success) {
                return res.status(400).json({ 
                    error: "auth.errors.captchaVerificationFailed" 
                });
            }

            delete req.body.captchaToken;
            
            await logActivity(
                null, 'auth', 'captcha_verified', 'low',
                `Captcha verified from IP: ${req.ip}`,
                req,
                getDeviceFingerprint(req),
                { captcha_provider: captcha.provider }
            );

            next();
        } catch (error) {
            console.error("Captcha verification error:", error);
            return res.status(500).json({ 
                error: "auth.errors.captchaVerificationFailed" 
            });
        }
    };

    /**
     * Inject security context (IP, user agent, device fingerprint) into the request object for later use in controllers and services.
     */
    injectSecurityContext = (req: Request, res: Response, next: NextFunction) => {
        (req as any).securityContext = {
            ipAddress: req.ip || '',
            userAgent: req.headers['user-agent'] || '',
            deviceFingerprint: getDeviceFingerprint(req)
        };
        next();
    };

    /**
     * Verify session cookie, check account status (blocked/suspended), and attach user/session to res.locals.
     */
    verifySession = async (req: Request, res: Response, next: NextFunction) => {
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

        // verify account status (blocked/suspended) and auto-clear expired suspension
        try {
            const accountStatus = await this.checkAccountStatus(user.id);
            
            if (accountStatus.isBlocked) {
                await lucia.invalidateSession(session.id);
                res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
                
                await logActivity(
                    user.id, 'auth', 'session_revoked_blocked', 'critical',
                    'Session revoked - Account blocked',
                    req,
                    getDeviceFingerprint(req)
                );
                
                return res.status(403).json({ error: "auth.errors.accessDenied" });
            }

            if (accountStatus.isSuspended) {
                await lucia.invalidateSession(session.id);
                res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
                
                await logActivity(
                    user.id, 'auth', 'session_revoked_suspended', 'high',
                    'Session revoked - Account suspended',
                    req,
                    getDeviceFingerprint(req)
                );
                
                return res.status(403).json({ error: "auth.errors.accessDenied" });
            }
        } catch (error) {
            console.error("Account status check error:", error);
            await lucia.invalidateSession(session.id);
            return res.status(500).json({ error: "auth.errors.internalServer" });
        }

        res.locals.user = user;
        res.locals.session = session;
        return runWithContext({ userId: user?.id ?? null }, next);
    };

    /**
     * Check if a user's account is blocked or suspended.  
     * If suspended, also check if the suspension has expired and auto-clear it.
     * If blocked or currently suspended, return the status for further handling (e.g. session invalidation).
     */
    private async checkAccountStatus(userId: string) {
        const security = await pool.query(
            "SELECT is_blocked, is_suspended, suspended_until FROM user_security WHERE user_id = $1",
            [userId]
        );

        if (security.rows.length === 0) {
            return { isBlocked: false, isSuspended: false };
        }

        const data = security.rows[0];
        let isSuspended = false;

        if (data.is_suspended && data.suspended_until) {
            if (new Date(data.suspended_until) > new Date()) {
                isSuspended = true;
            } else {
                // 自動清除過期 suspension
                await pool.query(
                    "UPDATE user_security SET is_suspended = FALSE, suspended_until = NULL WHERE user_id = $1",
                    [userId]
                );
            }
        }

        return {
            isBlocked: data.is_blocked,
            isSuspended: isSuspended
        };
    }

    /**
     * Middleware to require authentication (session must be valid and account must be active).
     */
    requireAuth = (req: Request, res: Response, next: NextFunction) => {
        if (!res.locals.session) {
            return res.status(401).json({ error: "auth.errors.unauthorized" });
        }
        next();
    };

    /**
     * Middleware to authenticate session and attach user/session to req for downstream use in controllers/services.
     */
    authenticateSession = (req: Request, res: Response, next: NextFunction) => {
        if (!res.locals.session || !res.locals.user) {
            return res.status(401).json({ error: "auth.errors.unauthorized" });
        }
        (req as any).user = res.locals.user;
        (req as any).session = res.locals.session;
        next();
    };

    /**
     * Verify that the authenticated user has the required permission (by scope, action, resource).
     */
    requirePermission = (scope: string, action: string, resource: string) => {
        return async (req: Request, res: Response, next: NextFunction) => {
            if (!res.locals.session || !res.locals.user) {
                return res.status(401).json({ error: "auth.errors.unauthorized" });
            }

            try {
                const userId = res.locals.user.id;

                const hasPermission = await this.rbacService.hasPermission(userId, scope, action, resource);

                if (!hasPermission) {
                    return res.status(403).json({ 
                        error: `auth.errors.permissionDenied`
                    });
                }
                
                return next();
            } catch (error) {
                console.error("Permission verification error", error);
                return res.status(500).json({ error: "auth.errors.internal" });
            }
        };
    };
}

/**
 * Export middleware functions for use in route definitions.
 */
const authMiddleware = new AuthMiddleware();

export const checkIpBlocked = authMiddleware.checkIpBlocked;
export const verifyCaptcha = authMiddleware.verifyCaptcha;
export const injectSecurityContext = authMiddleware.injectSecurityContext;
export const verifySession = authMiddleware.verifySession;
export const requireAuth = authMiddleware.requireAuth;
export const authenticateSession = authMiddleware.authenticateSession;
export const requirePermission = authMiddleware.requirePermission;
