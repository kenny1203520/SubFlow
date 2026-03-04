import { BaseController } from './BaseController';
import { AuthError, AuthService } from "../services/AuthService";
import express from 'express';
import { z } from 'zod';
import { getDeviceFingerprint } from '../utils/audit';
import { lucia } from '../auth/lucia';
import { SystemSettingService } from '../services/SystemSettingService';

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
    private static authService = new AuthService();
    private static systemSettingService = new SystemSettingService();

    register() {

    }

    // Validation Schemas
    private static verifyCaptcha = async (token: string, ip: string) => {
        const captcha = await this.systemSettingService.getCaptchaSettings();
        if (captcha.provider === "none") return true;

        if (!captcha.secretKey) {
            console.warn("CAPTCHA_SECRET_KEY is missing. CAPTCHA validation skipped but could be a security risk.");
            return true; 
        }

        if (!token) return false;

        let verifyUrl = "";
        if (captcha.provider === "turnstile") {
            verifyUrl = "https://challenges.cloudflare.com/turnstile/v0/siteverify";
        } else if (captcha.provider === "recaptcha") {
            verifyUrl = "https://www.google.com/recaptcha/api/siteverify";
        } else {
            return true; // Unknown provider
        }

        try {
            const formData = new URLSearchParams();
            formData.append("secret", captcha.secretKey);
            formData.append("response", token);
            formData.append("remoteip", ip);

            const response = await fetch(verifyUrl, {
                method: "POST",
                body: formData,
                headers: { "Content-Type": "application/x-www-form-urlencoded" }
            });

            const data = await response.json();
            return data.success;
        } catch (e) {
            console.error("Captcha verification error", e);
            return false;
        }
    };

    private static buildContext(req: express.Request) {
        return {
            ip_address: req.ip || '',
            user_agent: (req.headers['user-agent'] || '') as string,
            device_fingerprint: getDeviceFingerprint(req),
            method: req.method,
            path: req.originalUrl
        };
    }

    private static handleError(res: express.Response, error: any) {
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

    static async getCaptchaConfig(req: express.Request, res: express.Response) {
        try {
            const config = await this.authService.getCaptchaConfig();
            res.json(config);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    static async signupHandler(req: express.Request, res: express.Response) {
        try {
            const { username, email, password, captchaToken } = this.authService.signupSchema.parse(req.body);
            const ctx = this.buildContext(req);
            const result = await this.authService.signupHandler(username, email, password, { captchaT: captchaToken, ...ctx });

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

    static async signinHandler(req: express.Request, res: express.Response) {
        try {
            const { username, password, captchaToken } = this.authService.signinSchema.parse(req.body);

            const isCaptchaValid = await this.verifyCaptcha(captchaToken || "", req.ip || "");
            if (!isCaptchaValid) {
                return res.status(400).json({ success: false, message: "auth.errors.captchaFailed" });
            }

            const ctx = this.buildContext(req);
            const result = await this.authService.signinHandler(username, password, { captchaT: captchaToken, ...ctx });

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

    static async signin2faHandler(req: express.Request, res: express.Response) {
        try {
            const { userId, code, captchaToken } = req.body;

            const isCaptchaValid = await this.verifyCaptcha(captchaToken || "", req.ip || "");
            if (!isCaptchaValid) {
                return res.status(400).json({ success: false, message: "auth.errors.captchaFailed" });
            }

            if (typeof userId !== "string" || typeof code !== "string") {
                return res.status(400).json({ success: false, message: "auth.errors.invalidInput" });
            }

            const ctx = this.buildContext(req);
            const result = await this.authService.signin2faHandler(userId, code, { captchaT: captchaToken, ...ctx });

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

    static async signoutHandler(req: express.Request, res: express.Response) {
        try {
            const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "") || "";
            const ctx = this.buildContext(req);
            const result = await this.authService.signoutHandler(sessionId, { ...ctx });
            if (result.success === false) {
                return res.status(400).json({ success: result.success, message: result.message || "auth.errors.signoutFailed" });
            }
            res.setHeader("Set-Cookie", result.sessionCookie?.serialize() || "");
            return res.status(200).json({ success: result.success });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    static async userHandler(req: express.Request, res: express.Response) {
        try {
            const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "") || "";
            const ctx = this.buildContext(req);
            const result = await this.authService.userHandler(sessionId, { ...ctx });

            if (result.sessionCookie) {
                res.setHeader("Set-Cookie", result.sessionCookie.serialize());
            }

            return res.status(200).json({
                ...result.user,
                system_roles: result.meta.system_roles,
                permissions: result.meta.permissions,
            });
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    static async verifyEmailHandler(req: express.Request, res: express.Response) {
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

    static async passwordResetHandler(req: express.Request, res: express.Response) {
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

    static async passwordResetTokenHandler(req: express.Request, res: express.Response) {
        try {
            const { token } = req.params;
            const { password, captchaToken } = this.authService.resetSchema.parse(req.body);
            const ctx = this.buildContext(req);
            const normalizedToken = Array.isArray(token) ? token[0] : token;
            const result = await this.authService.passwordResetTokenHandler(normalizedToken, password, { captchaT: captchaToken, ...ctx });
            return res.status(200).json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    static async changePasswordHandler(req: express.Request, res: express.Response) {
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
    static async generatePassKeyRegistrationOptions(req: express.Request, res: express.Response) {
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
    static async verifyPassKeyRegistration(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { credential, deviceName } = req.body;
            const result = await this.authService.verifyPassKeyRegistration(userId, credential, deviceName, req);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Generate PassKey authentication options
     */
    static async generatePassKeyAuthenticationOptions(req: express.Request, res: express.Response) {
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
    static async verifyPassKeyAuthentication(req: express.Request, res: express.Response) {
        try {
            const { assertion } = req.body;
            const result = await this.authService.verifyPassKeyAuthentication(assertion, req);
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
    static async listPassKeys(req: express.Request, res: express.Response) {
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
    static async deletePassKey(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const credentialId = Array.isArray(id) ? id[0] : id;
            const result = await this.authService.deletePassKey(userId, credentialId, req);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    // ===================== LDAP Authentication =====================

    /**
     * LDAP Login
     */
    static async ldapSignin(req: express.Request, res: express.Response) {
        try {
            const { username, password } = req.body;
            const result = await this.authService.ldapSignin(username, password, req);
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
    static async getSSOProviders(req: express.Request, res: express.Response) {
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
    static async initiateSSOLogin(req: express.Request, res: express.Response) {
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
    static async handleSSOCallback(req: express.Request, res: express.Response) {
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
            const result = await this.authService.handleSSOCallback(providerName, codeStr, req);
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
    static async getDevices(req: express.Request, res: express.Response) {
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
    static async trustDevice(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const deviceId = Array.isArray(id) ? id[0] : id;
            const result = await this.authService.trustDevice(userId, deviceId, req);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Revoke a device
     */
    static async revokeDevice(req: express.Request, res: express.Response) {
        try {
            const userId = req.user?.id;
            if (!userId) {
                return res.status(401).json({ message: 'auth.errors.notAuthenticated' });
            }

            const { id } = req.params;
            const deviceId = Array.isArray(id) ? id[0] : id;
            const result = await this.authService.revokeDevice(userId, deviceId, req);
            return res.json(result);
        } catch (error: any) {
            return this.handleError(res, error);
        }
    }

    /**
     * Get device notifications
     */
    static async getDeviceNotifications(req: express.Request, res: express.Response) {
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
    static async acknowledgeNotification(req: express.Request, res: express.Response) {
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
