import { BaseController } from './BaseController';
import express from 'express';
import { z } from 'zod';
import { lucia } from '../auth/lucia';
import { getDeviceFingerprint } from '../utils/audit';
import { AuthError, SecurityContext, AuthService } from "../services/AuthService";
import { Server, Socket } from 'socket.io';
import { SystemSettingService } from '../services/SystemSettingService';
import { authSocketEvents } from '../socket/events';

/**
 * AuthController handles advanced authentication methods:
 * - Password-based login
 * - 2FA (TOTP)
 * - WebAuthn/PassKey
 * - LDAP
 * - SSO (OAuth2/OIDC)
 * - Device Management
 */
export class AuthController extends BaseController {
    private authService: AuthService;
    private systemSettingService: SystemSettingService;

    constructor(io?: Server, socket?: Socket) {
        super(io, socket);
        this.authService = new AuthService();
        this.systemSettingService = new SystemSettingService();
    }

    register() {
        if (!this.socket) return;

        this.socket.on(authSocketEvents.USER, async (_payload: any, cb: (res: any) => void) => {
            try {
                const sessionId = this.socket?.data?.session?.id || "";
                const result = await this.authService.userHandler(sessionId, this.buildSocketContext());
                this.success(cb, result);
            } catch (error: any) {
                this.error(cb, error?.message || "auth.errors.internalServer");
            }
        });
    }

    private buildSocketContext(): SecurityContext {
        const ipAddress = this.socket?.handshake.address || '';
        const userAgent = String(this.socket?.handshake.headers['user-agent'] || '');
        const deviceFingerprint = `${ipAddress}|${userAgent}`;
        return { ipAddress, userAgent, deviceFingerprint };
    }

    async getCaptchaConfig(_req: express.Request, res: express.Response) {
        try {
            const settings = await this.systemSettingService.getCaptchaSettings();
            return res.status(200).json({
                provider: settings.provider,
                siteKey: settings.siteKey || null,
                enabled: settings.provider !== 'none'
            });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    private buildContext(req: express.Request): SecurityContext {
        return {
            ipAddress: req.ip || '',
            userAgent: (req.headers['user-agent'] || '') as string,
            deviceFingerprint: getDeviceFingerprint(req),
        };
    }

    private handleError(res: express.Response, error: any) {
        if (error instanceof z.ZodError) {
            return res.status(400).json({ message: "auth.errors.invalidInput", details: error.issues });
        }

        if (error instanceof AuthError) {
            if (error.sessionCookie) {
                res.setHeader("Set-Cookie", error.sessionCookie);
            }
            return res.status(error.statusCode).json({ message: error.message, details: error.details });
        }

        console.error(error);
        return res.status(500).json({ message: "auth.errors.internalServer" });
    }

    async signupHandler(req: express.Request, res: express.Response) {
        try {
            const { username, email, password } = this.authService.signupSchema.parse(req.body);
            const ctx = this.buildContext(req);
            const result = await this.authService.signupHandler(username, email, password, ctx);

            res.setHeader("Set-Cookie", result.sessionCookie?.serialize() || "");
            return res.status(201).json({
                success: true,
                user: result.user,
                message: result.message
            });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async signinHandler(req: express.Request, res: express.Response) {
        try {
            const { username, password } = this.authService.signinSchema.parse(req.body);

            const ctx = this.buildContext(req);
            const result = await this.authService.signinHandler(username, password, ctx );

            if (result.requires2FA) {
                return res.status(200).json({
                    success: true,
                    requires2FA: true,
                    userId: result.user?.id
                });
            }

            if (result.success === false) {
                return res.status(400).json({ success: result.success, message: result.message || "auth.errors.invalidCredentials" });
            }

            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }
            else {
                return res.status(500).json({ success: result.success, message: "auth.errors.sessionCreationFailed" });
            }

            return res.status(200).json({ success: result.success, user: result.user });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async signin2faHandler(req: express.Request, res: express.Response) {
        try {
            const { userId, code } = req.body;

            if (typeof userId !== "string" || typeof code !== "string") {
                return res.status(400).json({ success: false, message: "auth.errors.invalidInput" });
            }

            const ctx = this.buildContext(req);
            const result = await this.authService.signin2faHandler(userId, code, ctx);

            if (result.success === false) {
                return res.status(400).json({ success: result.success, message: result.message || "auth.errors.invalidTwoFactor" });
            }

            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }
            else {
                return res.status(500).json({ success: result.success, message: "auth.errors.sessionCreationFailed" });
            }

            return res.status(200).json({ success: result.success, user: result.user });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async signoutHandler(req: express.Request, res: express.Response) {
        try {
            const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "") || "";
            const ctx = this.buildContext(req);
            const result = await this.authService.signoutHandler(sessionId, ctx );
            if (result.success === false) {
                return res.status(400).json({ success: result.success, message: result.message || "auth.errors.signoutFailed" });
            }
            res.setHeader("Set-Cookie", result.sessionCookie?.serialize() || "");
            return res.status(200).json({ success: result.success });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async userHandler(req: express.Request, res: express.Response) {
        try {
            const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "") || "";
            const ctx = this.buildContext(req);
            const result = await this.authService.userHandler(sessionId, ctx );

            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }

            return res.status(200).json({
                ...result.user,
                system_roles: result.meta?.system_roles || [],
                permissions: result.meta?.permissions || [],
            });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async verifyEmailHandler(req: express.Request, res: express.Response) {
        try {
            const { token } = req.params;
            const ctx = this.buildContext(req);
            const normalizedToken = Array.isArray(token) ? token[0] : token;
            const result = await this.authService.verifyEmailHandler(normalizedToken, ctx);
            return res.status(200).json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async passwordResetHandler(req: express.Request, res: express.Response) {
        try {
            let { email } = req.body;
            if (typeof email !== "string" || !email.includes("@")) {
                return res.status(400).json({ message: "auth.errors.invalidEmail" });
            }
            email = email.toLowerCase().trim();

            const ctx = this.buildContext(req);
            const result = await this.authService.passwordResetHandler(email, ctx);
            return res.status(200).json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async passwordResetTokenHandler(req: express.Request, res: express.Response) {
        try {
            const { token } = req.params;
            const { password } = this.authService.resetSchema.parse(req.body);
            const ctx = this.buildContext(req);
            const normalizedToken = Array.isArray(token) ? token[0] : token;
            const result = await this.authService.passwordResetTokenHandler(normalizedToken, password, ctx);
            return res.status(200).json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    async changePasswordHandler(req: express.Request, res: express.Response) {
        try {
            const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "") || "";
            const { oldPassword, newPassword } = this.authService.changePasswordSchema.parse(req.body);
            const ctx = this.buildContext(req);
            const result = await this.authService.changePasswordHandler(sessionId, oldPassword, newPassword, ctx);
            return res.status(200).json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    // ===================== PassKey/WebAuthn =====================

    /**
     * Generate PassKey registration options
     */
    async generatePassKeyRegistrationOptions(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const options = await this.authService.generatePassKeyRegistrationOptions(userId);
            return res.json(options);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Verify and register PassKey
     */
    async verifyPassKeyRegistration(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { credential, deviceName } = req.body;
            const ctx = this.buildContext(req);
            const result = await this.authService.verifyPassKeyRegistration(userId, credential, deviceName, ctx);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Generate PassKey authentication options
     */
    async generatePassKeyAuthenticationOptions(req: express.Request, res: express.Response) {
        try {
            const { userId } = req.body; // Optional - for user-specific credentials
            const options = await this.authService.generatePassKeyAuthenticationOptions(userId);
            return res.json(options);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Verify PassKey authentication and create session
     */
    async verifyPassKeyAuthentication(req: express.Request, res: express.Response) {
        try {
            const { assertion } = req.body;
            const ctx = this.buildContext(req);
            const result = await this.authService.verifyPassKeyAuthentication(assertion, ctx);
            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }
            return res.json({ success: result.success, user: result.user });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * List user's PassKeys
     */
    async listPassKeys(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const result = await this.authService.listPassKeys(userId);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Delete a PassKey
     */
    async deletePassKey(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const credentialId = Array.isArray(id) ? id[0] : id;
            const ctx = this.buildContext(req);
            const result = await this.authService.deletePassKey(userId, credentialId, ctx);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    // ===================== LDAP Authentication =====================

    /**
     * LDAP Login
     */
    async ldapSignin(req: express.Request, res: express.Response) {
        try {
            const { username, password } = req.body;
            const ctx = this.buildContext(req);
            const result = await this.authService.ldapSignin(username, password, ctx);
            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }
            return res.json({ success: result.success, user: result.user });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    // ===================== SSO Authentication =====================

    /**
     * Get available SSO providers
     */
    async getSSOProviders(req: express.Request, res: express.Response) {
        try {
            const result = await this.authService.getSSOProviders();
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Initiate SSO login
     */
    async initiateSSOLogin(req: express.Request, res: express.Response) {
        try {
            const { provider } = req.params;
            const providerName = Array.isArray(provider) ? provider[0] : provider;
            const state = Math.random().toString(36).substring(7);

            const { authUrl } = await this.authService.initiateSSOLogin(providerName, state);

            // Store state in session for CSRF protection
            // In production, use Redis or database
            req.session = req.session || {};
            req.session.oauthState = state;

            return res.redirect(authUrl);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Handle SSO callback
     */
    async handleSSOCallback(req: express.Request, res: express.Response) {
        try {
            const { provider } = req.params;
            const { code, state } = req.query;

            if (!code) {
                return res.status(400).json({ message: 'auth.errors.missingCode' });
            }

            // Verify state (CSRF protection)
            // In production, verify against stored state
            // if (state !== req.session?.oauthState) {
            //     return res.status(400).json({ message: 'auth.errors.invalidState' });
            // }

            const providerName = Array.isArray(provider) ? provider[0] : provider;
            const codeStr = (Array.isArray(code) ? code[0] : code) as string || '';
            const ctx = this.buildContext(req);
            const result = await this.authService.handleSSOCallback(providerName, codeStr, ctx);
            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }
            return res.redirect(result.redirectUrl);
        } catch (error: any) {
            if (error instanceof AuthError) {
                return this.handleError(res, error);
            }
            const frontendUrl = process.env.FRONTEND_URL || 'http://localhost:5173';
            return res.redirect(`${frontendUrl}/auth/error?message=sso_failed`);
        }
    }

    // ===================== Device Management =====================

    /**
     * Get user's devices
     */
    async getDevices(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const result = await this.authService.getDevices(userId);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Trust a device
     */
    async trustDevice(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const deviceId = Array.isArray(id) ? id[0] : id;
            const ctx = this.buildContext(req);
            const result = await this.authService.trustDevice(userId, deviceId, ctx);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Revoke a device
     */
    async revokeDevice(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const deviceId = Array.isArray(id) ? id[0] : id;
            const ctx = this.buildContext(req);
            const result = await this.authService.revokeDevice(userId, deviceId, ctx);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Get device notifications
     */
    async getDeviceNotifications(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const includeRead = req.query.includeRead === 'true';
            const result = await this.authService.getDeviceNotifications(userId, includeRead);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Acknowledge device notification
     */
    async acknowledgeNotification(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const notificationId = Array.isArray(id) ? id[0] : id;
            const result = await this.authService.acknowledgeNotification(userId, notificationId);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }
}
