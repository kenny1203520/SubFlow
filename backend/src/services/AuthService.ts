import express from "express";
import { lucia } from "../auth/lucia";
import { Scrypt } from "oslo/password";
import { pool } from "../db";
import { MailService } from "../services/MailService";
import crypto from "crypto";
import { z } from "zod";
import { SystemSettingsService } from "../services/SystemSettingsService";
import { verify } from "otplib";
import { logActivity, getDeviceFingerprint } from "../utils/audit";
import { SecurityRepository } from "../repositories/SecurityRepository";
import { SecurityService } from "./SecurityService";
import { RBACService } from "./RBACService";
import { UserRepository } from "../repositories/UserRepository";
import { AuthRepository } from "../repositories/AuthRepository";
import { RBACRepository } from "../repositories/RBACRepository";
import { PassKeyService } from "./PassKeyService";
import { LDAPService } from "./LDAPService";
import { SSOService } from "./SSOService";
import { 
    getClientIpAddress, 
    parseUserAgent, 
    calculateLoginRiskScore 
} from "../utils/deviceFingerprint";

const scrypt = new Scrypt();

// Helper to get peppered password
const getPepperedPassword = (password: string) => {
    const authSecret = process.env.AUTH_SECRET;
    if (!authSecret) {
        throw new Error("AUTH_SECRET is not defined in environment variables");
    }
    return crypto.createHmac('sha256', authSecret).update(password).digest('hex');
};

export interface AuthResult {
    success: boolean;
    requires2FA?: boolean;
    user?: {
        id: string;
        username: string;
        email: string;
        avatar_url?: string;
        is_verified: boolean;
    };
    sessionCookie?: ReturnType<typeof lucia.createSessionCookie>;
    message?: string;
    details?: any;
    meta?: any;
}

export class AuthError extends Error {
    statusCode: number;
    details?: any;
    sessionCookie?: string;

    constructor(message: string, statusCode = 400, details?: any, sessionCookie?: string) {
        super(message);
        this.name = "AuthError";
        this.statusCode = statusCode;
        this.details = details;
        this.sessionCookie = sessionCookie;
    }
}

export class AuthService {
    private rbacRepo = new RBACRepository();
    private rbacService = new RBACService();
    private userRepo = new UserRepository();
    private authRepo = new AuthRepository();
    private securityRepo = new SecurityRepository();
    private securityService = new SecurityService();
    private passKeyService = new PassKeyService();
    private ldapService = new LDAPService();
    private ssoService = new SSOService();

    public signupSchema = z.object({
        username: z.string().min(3).max(255).regex(/^[a-zA-Z0-9_-]+$/, "auth.errors.usernameFormat"),
        email: z.email().max(255).regex(/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/, "auth.errors.invalidEmail").transform(v => v.toLowerCase()),
        password: z.string().min(8).max(255).regex(/[A-Z]/, "auth.errors.passwordUppercase").regex(/[0-9]/, "auth.errors.passwordNumber").regex(/[^A-Za-z0-9]/, "auth.errors.passwordSymbol"),
        captchaToken: z.string().optional(),
    });

    public signinSchema = z.object({
        username: z.string().max(255).regex(/^[a-zA-Z0-9_-]+$/, "auth.errors.usernameFormat"),
        password: z.string().max(255),
        captchaToken: z.string().optional(),
    });

    public signin2FASchema = z.object({
        userId: z.uuid("auth.errors.invalidUserId"),
        code: z.string().min(6).max(8).regex(/^[0-9]+$/, "auth.errors.invalidTwoFactorCode"),
    });

    public resetSchema = z.object({
        password: z.string().min(8).max(255).regex(/[A-Z]/, "auth.errors.passwordUppercase").regex(/[0-9]/, "auth.errors.passwordNumber").regex(/[^A-Za-z0-9]/, "auth.errors.passwordSymbol"),
        captchaToken: z.string().optional(),
    });

    public changePasswordSchema = z.object({
        oldPassword: z.string().max(255),
        newPassword: z.string().min(8).max(255).regex(/[A-Z]/, "auth.errors.passwordUppercase").regex(/[0-9]/, "auth.errors.passwordNumber").regex(/[^A-Za-z0-9]/, "auth.errors.passwordSymbol"),
    });
    
    getCaptchaConfig(): Promise<any> {
        const captchaProvider =  process.env.CAPTCHA_PROVIDER || "none"
        const captchaSiteKey = process.env.CAPTCHA_SITE_KEY || ""
        return Promise.resolve({
            captchaProvider: captchaProvider,
            captchaSiteKey: captchaSiteKey
        });
    }

    /**
     * Signup handler to create a new user, send verification email, and auto-login
     * With device tracking and risk-based notifications
     * @param un 
     * @param eMail 
     * @param pass 
     * @param data 
     * @returns 
     */
    async signupHandler(
        un: string, eMail: string, pass: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_username: un }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const { username, email, password, captchaToken } = this.signupSchema.parse({
                username: un,
                email: eMail,
                password: pass,
                captchaToken: data?.captchaT
            });

            const passwordHash = await scrypt.hash(getPepperedPassword(password));

            // Create user
            const user = await this.userRepo.createUser(username, email, passwordHash, false);
            const userId = user.id;

            // Generate Verification Token
            const token = crypto.randomBytes(32).toString("hex");
            const tokenHash = crypto.createHash("sha256").update(token).digest("hex");
            const expiresAt = new Date(Date.now() + 1000 * 60 * 60 * 24); // 24 hours

            // Create email verification record
            await this.authRepo.createEmailVerificationToken(userId, email, tokenHash, expiresAt);
            
            // Initialize User Security (with local auth provider)
            await this.securityRepo.createSecuritySettings(userId);
            await this.securityRepo.updateAuthProvider(userId, 'local', '');
            
            // Assign 'Guest' role
            const guestRole = await this.rbacRepo.getSystemRoleByName("Guest");
            if (guestRole) await this.rbacRepo.assignRoleToUser(userId, guestRole.id);

            // Send verification email
            await MailService.sendVerificationEmail(email, token);

            // Extract device information
            const fingerprint = data?.device_fingerprint || '';
            const ipAddress = data?.ip_address || data?.req?.ip || '';
            const userAgent = data?.user_agent || data?.req?.headers['user-agent'] || '';

            // Create session and cookie for auto-login after signup
            const session = await lucia.createSession(userId, {
                ip_address: ipAddress,
                user_agent: userAgent,
                device_fingerprint: fingerprint
            });
            const sessionCookie = lucia.createSessionCookie(session.id);

            // Register device using new SecurityService method
            const { device, isNewDevice } = await this.securityService.registerOrUpdateDevice(
                userId,
                {
                    fingerprint,
                    userAgent,
                    ipAddress
                }
            );

            // Link device to session
            await this.authRepo.createLoginHistory(
                userId, 
                session.id, 
                new Date(),
                ipAddress, 
                userAgent,
                fingerprint,
                device.id, 
                'success'
            );

            // Log signup activity
            await logActivity(
                userId, 'auth', 'signup', 'low',
                'User signed up and auto-logged in',
                data?.req, fingerprint, 
                { username, email, device_id: device.id }
            );

            // Send new device notification (for first device after signup, always send)
            if (isNewDevice) {
                await this.securityService.notifyNewDeviceLogin(userId, device.id, {
                    deviceName: device.device_name || 'Unknown Device',
                    ipAddress,
                    location: device.location
                });
            }

            return {
                success: true,
                user: {
                    id: userId,
                    username: username,
                    email: email,
                    is_verified: false
                },
                sessionCookie,
                message: "Signup successful. Please verify your email.",
            }
        } catch (error: any) {
            console.error("Signup error:", error);
            await logActivity(
                null, 'auth', 'signup_failed', 'high',
                `Signup failed for user: ${un}`,
                data?.req, data?.device_fingerprint, 
                { error: error.message }
            );
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    /**
     * Signin handler with enhanced device tracking and risk-based authentication
     * @param un 
     * @param pass 
     * @param data 
     * @returns 
     */
    async signinHandler(
        un: string, pass: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_username: un }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const { username, password, captchaToken } = this.signinSchema.parse({
                username: un,
                password: pass,
                captchaToken: data?.captchaT
            });

            // Extract device information
            const fingerprint = data?.device_fingerprint || '';
            const ipAddress = data?.ip_address || data?.req?.ip || '';
            const userAgent = data?.user_agent || data?.req?.headers['user-agent'] || '';
            const req = data?.req;

            // Fetch user by username
            const user = await this.userRepo.getByUsername(username);

            if (!user) {
                await logActivity(
                    null, 'auth', 'login_failed', 'medium',
                    `Failed login for user: ${username}`,
                    req, fingerprint, 
                    { attempted_username: username }
                );
                // Always return the same error message for security
                return { 
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }
            
            // Check Account Lockout Status
            const securitySettings = await this.securityRepo.getSecuritySettings(user.id);
            if (!securitySettings) await this.securityRepo.createSecuritySettings(user.id);

            // Check if account is blocked or suspended before verifying password
            if (await this.securityRepo.isUserBlocked(user.id)) {
                await logActivity(
                    user.id, 'auth', 'login_blocked', 'critical',
                    'Login attempted on blocked account',
                    req, fingerprint, 
                    { attempted_username: username }
                );
                return {
                    success: false,
                    message: "auth.errors.accountBlocked" 
                };
            }

            if (await this.securityRepo.isUserSuspended(user.id)) {
                await logActivity(
                    user.id, 'auth', 'login_suspended', 'high',
                    'Login attempted on suspended account',
                    req, fingerprint, 
                    { attempted_username: username }
                );
                return { 
                    success: false,
                    message: "auth.errors.accountSuspended" 
                };
            }

            // Check if user is using the correct auth provider (e.g. local vs ldap vs sso)
            const authProvider = await this.securityRepo.getAuthProvider(user.id);
            if (authProvider && authProvider.auth_provider !== 'local') {
                await logActivity(
                    user.id, 'auth', 'login_failed', 'medium',
                    `Login attempted with local provider on account registered with ${authProvider.auth_provider}`,
                    req, fingerprint, 
                    { attempted_username: username }
                );
                return {
                    success: false,
                    message: `auth.errors.use${authProvider.auth_provider}`
                }
            }

            // Calculate risk score before password validation
            const existingDevices = await this.securityRepo.getDevices(user.id);
            const previousDevice = existingDevices.find(d => d.device_fingerprint === fingerprint);
            const isNewDevice = !previousDevice;
            const isNewLocation = !existingDevices.some(d => 
                d.ip_address && ipAddress && d.ip_address.split('.').slice(0, 3).join('.') === ipAddress.split('.').slice(0, 3).join('.')
            );
            
            const accountAgeInDays = user.created_at 
                ? Math.floor((Date.now() - user.created_at.getTime()) / (1000 * 60 * 60 * 24))
                : 0;
            const lastDevice = existingDevices
                .filter(d => d.last_active_at)
                .sort((a, b) => 
                    new Date(b.last_active_at!).getTime() - new Date(a.last_active_at!).getTime()
                )[0];
            const timeSinceLastLoginHours = lastDevice && lastDevice.last_active_at
                ? Math.floor((Date.now() - new Date(lastDevice.last_active_at).getTime()) / (1000 * 60 * 60))
                : 99999;
            
            const fingerprintSimilarity = previousDevice ? 1.0 : 0.0;
            
            const loginRiskScore = calculateLoginRiskScore({
                isNewDevice,
                isNewLocation,
                fingerprintSimilarity,
                failedAttemptsRecently: securitySettings?.failed_login_attempts || 0,
                accountAge: accountAgeInDays,
                timeSinceLastLogin: timeSinceLastLoginHours
            });

            const validPassword = await scrypt.verify(user.password_hash, getPepperedPassword(password));
            if (!validPassword) {
                // Increment failed attempts
                const lockoutCfg = await SystemSettingsService.getSetting('security.auth_lockout');
                const maxAttempts   = lockoutCfg?.maxFailedAttempts ?? 5;
                const lockoutDurationMins = lockoutCfg?.lockoutDurationMins ?? 720;
                
                await this.securityRepo.incrementFailedLoginAttempts(user.id);
                const newAttempts = await this.securityRepo.getFailedLoginAttempts(user.id);
                if (newAttempts >= maxAttempts) {
                    const suspendUntil = new Date(Date.now() + lockoutDurationMins * 60 * 1000);
                    await this.securityRepo.suspendUser(user.id, suspendUntil);
                    await logActivity(
                        user.id, 'auth', 'account_lockout', 'critical',
                        `Account locked out after ${newAttempts} failed attempts`,
                        req, fingerprint, 
                        { attempted_username: username, failed_attempts: newAttempts }
                    );
                }
                
                await logActivity(
                    user.id, 'auth', 'login_failed', 'medium',
                    `Invalid password for user: ${username}`,
                    req, fingerprint,
                    { attempted_username: username, risk_score: loginRiskScore }
                );
                
                // Log security event
                await this.authRepo.createLoginHistory(
                    user.id, '', new Date(),
                    ipAddress,
                    userAgent,
                    fingerprint,
                    '', // no device_id yet
                    'failed', 
                    'Invalid password'
                );

                return { 
                    success: false,
                    message: "auth.errors.invalidCredentials" 
                };
            }

            // Reset failed attempts on success (or partial success like 2FA)
            await this.securityRepo.resetFailedLoginAttempts(user.id);

            // Check if 2FA is enabled or required by risk score
            const shouldRequire2FA = securitySettings?.two_factor_enabled || loginRiskScore >= 40;
            
            if (shouldRequire2FA && securitySettings?.two_factor_enabled) {
                // Validate that we have a secret to check against
                if (!securitySettings.two_factor_secret) {
                    console.error("2FA enabled but no secret found for user", user.id);
                    return {
                        success: false,
                        message: "auth.errors.securityConfig"
                    };
                }
                
                await logActivity(
                    user.id, 'auth', '2fa_required', 'low', 
                    `2FA code required for login (risk score: ${loginRiskScore})`, 
                    req, fingerprint,
                    { username, risk_score: loginRiskScore }
                );
                
                return {
                    success: true,
                    requires2FA: true,
                    user: {
                        id: user.id,
                        username: user.username,
                        email: user.email,
                        is_verified: user.is_verified ?? false
                    }
                };
            }

            // Register or update device
            const { device, isNewDevice: isNewDeviceFlag } = await this.securityService.registerOrUpdateDevice(
                user.id,
                {
                    fingerprint,
                    userAgent,
                    ipAddress
                }
            );

            // Create session
            const session = await lucia.createSession(user.id, {
                ip_address: ipAddress,
                user_agent: userAgent,
                device_fingerprint: fingerprint
            });
            const sessionCookie = lucia.createSessionCookie(session.id);
            
            await logActivity(
                user.id, 'auth', 'login', 'info', 
                'User logged in', 
                req, fingerprint, 
                { username, session_id: session.id, device_id: device.id, is_new_device: isNewDevice, risk_score: loginRiskScore }
            );

            // Link device to session
            await this.authRepo.createLoginHistory(
                user.id, 
                session.id, 
                new Date(),
                ipAddress,
                userAgent,
                fingerprint,
                device.id, 
                'success',
                undefined
            );

            // Send notification for new device or suspicious activity
            if (isNewDeviceFlag) {
                await this.securityService.notifyNewDeviceLogin(user.id, device.id, {
                    deviceName: device.device_name || 'Unknown Device',
                    ipAddress,
                    location: device.location
                });
            } else if (loginRiskScore >= 30) {
                // High risk login from known device
                const userAgentInfo = parseUserAgent(userAgent);
                await MailService.sendSuspiciousActivityAlert(
                    user.email,
                    {
                        activityType: 'High Risk Login Detected',
                        details: `Login from ${userAgentInfo.browser} on ${userAgentInfo.os} with risk score ${loginRiskScore}`,
                        ipAddress: ipAddress,
                        timestamp: new Date()
                    }
                );
            }

            return {
                success: true,
                user: {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    is_verified: user.is_verified ?? false
                },
                sessionCookie,
            };
        } catch (error: any) {
            console.error("Signin error:", error);
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Signin error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack, attempted_username: un }
            );
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    async signin2faHandler(
        uId: string, co: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_user_id: uId }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const { userId, code } = this.signin2FASchema.parse({
                userId: uId,
                code: co,
                captchaT: data?.captchaT
            });
            
            if (typeof userId !== "string" || typeof code !== "string") {
                return {
                    success: false,
                    message: "auth.errors.invalidInput"
                }
            }

            // fetch user and secret
            const user = await this.userRepo.getById(userId);
            if (!user) {
                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const securitySettings = await this.securityRepo.getSecuritySettings(userId);

            if (!securitySettings || !securitySettings.two_factor_enabled || !securitySettings.two_factor_secret) {
                return {
                    success: false,
                    message: "auth.errors.twoFactorNotEnabled"
                };
            }

            // TOTP or Backup Code verification
            let isValid = false;
            let usedBackupCode = false;

            if (code.length === 8 && securitySettings.backup_codes && securitySettings.backup_codes.length > 0) {
                for (let i = 0; i < securitySettings.backup_codes.length; i++) {
                    const isMatch = await scrypt.verify(securitySettings.backup_codes[i], code);
                    if (isMatch) {
                        isValid = true;
                        usedBackupCode = true;
                        // Remove used backup code from array and save
                        const newBackupCodes = [...securitySettings.backup_codes];
                        newBackupCodes.splice(i, 1);
                        await this.securityRepo.updateBackupCodes(userId, newBackupCodes);
                        break;
                    }
                }
            } else {
                // @ts-ignore
                isValid = verify({ token: code, secret: securitySettings.two_factor_secret });
            }

            const fingerprint = data?.device_fingerprint || getDeviceFingerprint(data?.req);
            const ipAddress = data?.ip_address || data?.req?.ip || '';
            const userAgent = data?.user_agent || data?.req?.headers['user-agent'] || '';
            
            if (!isValid) {
                await logActivity(
                    userId, 'auth', '2fa_failed', 'high',
                    'Invalid 2FA code', 
                    data?.req, fingerprint, 
                    { user_id: userId, used_backup_code: code.length === 8 }
                );
                await logActivity(
                    userId,
                    'auth',
                    '2fa_failed',
                    'high',
                    'Invalid 2FA code',
                    data?.req,
                    fingerprint,
                    { used_backup_code: code.length === 8 }
                );
                return {
                    success: false,
                    message: "auth.errors.invalidTwoFactor"
                };
            }
            const session = await lucia.createSession(user.id, {
                ip_address: ipAddress,
                user_agent: userAgent,
                device_fingerprint: fingerprint,
            });
            const sessionCookie = lucia.createSessionCookie(session.id);
            
            await logActivity(
                user.id, 'auth', 'login_2fa', 'info', 
                `User logged in with 2FA${usedBackupCode ? ' (Backup Code)' : ''}`, 
                data?.req, fingerprint, 
                { username: user.username, session_id: session.id, used_backup_code: usedBackupCode }
            );

            return {
                success: true,
                user: {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    avatar_url: user.avatar_url,
                    is_verified: user.is_verified ?? false
                },
                sessionCookie,
            };

        } catch (error: any) {
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `2FA error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack, attempted_user_id: uId }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    async signoutHandler(
        sessionId: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {
        if (!sessionId) {
            return {
                success: false,
                message: "auth.errors.notAuthenticated"
            }
        }

        const ipAddress = data?.ip_address || data?.req?.ip || '';
        const userAgent = data?.user_agent || data?.req?.headers['user-agent'] || '';
        const fingerprint = data?.device_fingerprint || getDeviceFingerprint(data?.req);

        const { session } = await lucia.validateSession(sessionId);
        if (session) {
            await logActivity(
                session.userId, 'auth', 'signout', 'low', 
                'User signed out', 
                data?.req, fingerprint, 
                { session_id: sessionId, ip_address: ipAddress, user_agent: userAgent }
            );
            await lucia.invalidateSession(session.id);
        }
        
        return {
            success: true,
            message: "Signed out",
            sessionCookie: lucia.createBlankSessionCookie()
        };
    }

    async userHandler(
        sessionId: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {
        
        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_session_id: sessionId}
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }
            
            if (!sessionId) {
                return {
                    success: false,
                    message: "auth.errors.notAuthenticated"
                };
            }

            const { session, user } = await lucia.validateSession(sessionId);
            if (!session) {
                return {
                    success: false,
                    message: "auth.errors.notAuthenticated"
                };
            }

            if (session && session.fresh) {
                const sessionCookie = lucia.createSessionCookie(session.id);
                return {
                    success: true,
                    message: "Session refreshed",
                    sessionCookie: sessionCookie
                };
            }

            // Fetch system roles and flattened permissions for this user
            const systemRoles = await this.rbacService.getUserRoles(user.id);
            const permissions = await this.rbacService.getUserPermissions(user.id);

            return {
                success: true,
                user: {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    avatar_url: user.avatar_url,
                    is_verified: user.is_verified
                },
                meta: {
                    user: user,
                    system_roles: systemRoles,
                    permissions: permissions,
                }
            };
        } catch (error: any) {
            await logActivity(
                null, 'auth', 'system_error', 'high',
                `User handler error: ${error.message}`,
                data?.req, data?.device_fingerprint,
                { error: error.message, stack: error.stack, attempted_session_id: sessionId }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    async verifyEmailHandler(
        token: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {
        
        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_token: token }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const tokenHash = crypto.createHash("sha256").update(token as string).digest("hex");

            try {

                const verificationToken = await this.authRepo.getEmailVerificationTokenByTokenHash(tokenHash);

                if (!verificationToken) {
                    return {
                        success: false,
                        message: "auth.errors.invalidToken"
                    };
                }

                // Check if token is expired
                if (verificationToken.expires_at < new Date()) {
                    await this.authRepo.deleteEmailVerificationToken(verificationToken.id);
                    return {
                        success: false,
                        message: "auth.errors.expiredToken"
                    };
                }
                await this.userRepo.updateVerificationStatus(verificationToken.user_id, true);
                await this.authRepo.deleteEmailVerificationTokensByUserId(verificationToken.user_id);

                await logActivity(
                    verificationToken.user_id, 'auth', 'email_verified', 'low', 
                    'Email verified successfully', 
                    data?.req, data?.device_fingerprint, 
                    { email: verificationToken.email, user_id: verificationToken.user_id }
                );

                // Promote user from 'Guest' to 'User' role
                const guestRole = await this.rbacRepo.getSystemRoleByName("Guest");
                const userRole = await this.rbacRepo.getSystemRoleByName("User");
                await this.rbacRepo.assignRoleToUser(verificationToken.user_id, userRole?.id || '');
                await this.rbacRepo.removeRoleFromUser(verificationToken.user_id, guestRole?.id || '');

                return { 
                    success: true, 
                    message: "Email verified successfully." 
                };
            } catch (error: any) {
                await logActivity(
                    null, 'auth', 'system_error', 'high', 
                    `Email verification error: ${error.message}`, 
                    data?.req, data?.device_fingerprint, 
                    { error: error.message, stack: error.stack, params: data?.req?.params }
                );
                console.error(error);
                return { 
                    success: false, 
                    message: "auth.errors.internalServer" 
                };
            }
        } catch (error: any) {
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Email verification handler error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    async passwordResetHandler(
        eMail: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {
        
        if (typeof eMail !== "string" || !eMail.includes("@")) {
            return {
                success: false,
                message: "auth.errors.invalidEmail"
            };
        }
        const email = z.email().parse(eMail).toLowerCase(); // Validate email format

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_email: email }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const user = await this.userRepo.getByEmail(email);
            
            // Always return success to prevent enumeration
            if (!user) {
                await logActivity(
                    null, 'auth', 'reset_request_unknown', 'low', 
                    `Reset requested for non-existent: ${email}`, 
                    data?.req, data?.device_fingerprint, 
                    { attempted_email: email }
                );
                return {
                    success: true, 
                    message: "If account exists, email sent."
                };
            }

            const token = crypto.randomBytes(32).toString("hex");
            const tokenHash = crypto.createHash("sha256").update(token).digest("hex");
            const expiresAt = new Date(Date.now() + 1000 * 60 * 60); // 1 hour

            await this.authRepo.createPasswordResetToken(user.id, tokenHash, expiresAt);

            await MailService.sendPasswordResetEmail(email, token);
            await logActivity(
                user.id, 'auth', 'reset_request', 'info', 
                'Password reset requested', 
                data?.req, data?.device_fingerprint, 
                { email: user.email }
            );

            return { success: true, message: "If account exists, email sent." };
        } catch (error: any) {
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Password reset request error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack, attempted_email: email }
            );
            console.error(error);
            return { success: false, message: "auth.errors.internalServer" };
        }
    }

    async passwordResetTokenHandler(
        token: string, pass: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_token: token }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }
            
            const { password, captchaToken } = this.resetSchema.parse({
                password: pass,
                captchaToken: data?.captchaT
            });

            const tokenHash = crypto.createHash("sha256").update(token as string).digest("hex");

            // Fetch token and validate
            const resetToken = await this.authRepo.getPasswordResetTokenByTokenHash(tokenHash);
            if (!resetToken) {
                return {
                    success: false,
                    message: "Invalid or expired token"
                };
            }

            // Get user
            const user = await this.userRepo.getById(resetToken.user_id);
            if (!user) {
                return {
                    success: false,
                    message: "Invalid or expired token" // Don't reveal that user doesn't exist
                };
            }

            // Update password and invalidate token
            const passwordHash = await scrypt.hash(getPepperedPassword(password));
            await this.userRepo.updatePassword(user.id, passwordHash);
            await this.authRepo.markPasswordResetTokenUsed(resetToken.id);
            
            await logActivity(
                resetToken.user_id, 'auth', 'password_reset', 'high', 
                'Password reset successfully', 
                data?.req, data?.device_fingerprint, 
                { user_id: resetToken.user_id }
            );

            return { 
                success: true, 
                message: "Password updated." 
            };

        } catch (error: any) {
            if (error instanceof z.ZodError) {
                return { 
                    success: false, 
                    message: "auth.errors.invalidInput", 
                    details: error.issues 
                };
            }
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Password reset execution error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack, token: token }
            );
            console.error(error);
            return { success: false, message: "auth.errors.internalServer" };
        }
    }

    async generatePassKeyRegistrationOptions(userId: string) {
        const user = await this.userRepo.getById(userId);
        if (!user) {
            throw new AuthError("auth.errors.userNotFound", 404);
        }

        return this.passKeyService.generateRegistrationOptions(userId, user.username, user.email);
    }

    async verifyPassKeyRegistration(
        userId: string,
        credential: any,
        deviceName: string | undefined,
        req: express.Request
    ) {
        if (!credential) {
            throw new AuthError("auth.errors.missingCredential", 400);
        }

        try {
            const passkeyCredential = await this.passKeyService.verifyRegistration(
                userId,
                credential,
                deviceName || "PassKey Device"
            );

            await logActivity(
                userId,
                "auth",
                "passkey_registered",
                "info",
                "User registered a new PassKey",
                req,
                getDeviceFingerprint(req)
            );

            return {
                success: true,
                message: "PassKey registered successfully",
                credential: {
                    id: passkeyCredential.id,
                    deviceName: passkeyCredential.device_name
                }
            };
        } catch (error: any) {
            await logActivity(
                userId,
                "auth",
                "passkey_registration_failed",
                "medium",
                `PassKey registration failed: ${error.message}`,
                req,
                getDeviceFingerprint(req)
            );
            throw new AuthError(error.message || "auth.errors.registrationFailed", 400);
        }
    }

    async generatePassKeyAuthenticationOptions(userId?: string) {
        return this.passKeyService.generateAuthenticationOptions(userId);
    }

    async verifyPassKeyAuthentication(assertion: any, req: express.Request) {
        if (!assertion) {
            throw new AuthError("auth.errors.missingAssertion", 400);
        }

        try {
            const { userId } = await this.passKeyService.verifyAuthentication(assertion);

            const user = await this.userRepo.getById(userId);
            if (!user) {
                throw new AuthError("auth.errors.userNotFound", 404);
            }

            if (await this.securityRepo.isUserBlocked(userId)) {
                throw new AuthError("auth.errors.accountBlocked", 403);
            }

            if (await this.securityRepo.isUserSuspended(userId)) {
                throw new AuthError("auth.errors.accountSuspended", 403);
            }

            const fingerprint = getDeviceFingerprint(req);
            const ipAddress = getClientIpAddress(req);
            const userAgent = String(req.headers["user-agent"] || "");

            const session = await lucia.createSession(userId, {
                ip_address: ipAddress,
                user_agent: userAgent,
                device_fingerprint: fingerprint
            });

            await logActivity(
                userId,
                "auth",
                "passkey_login",
                "info",
                "User logged in with PassKey",
                req,
                fingerprint
            );

            return {
                success: true,
                sessionCookie: lucia.createSessionCookie(session.id),
                user: {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    is_verified: user.is_verified
                }
            };
        } catch (error: any) {
            if (error instanceof AuthError) {
                throw error;
            }

            await logActivity(
                null,
                "auth",
                "passkey_login_failed",
                "medium",
                `PassKey authentication failed: ${error.message}`,
                req,
                getDeviceFingerprint(req)
            );
            throw new AuthError(error.message || "auth.errors.authenticationFailed", 400);
        }
    }

    async listPassKeys(userId: string) {
        const credentials = await this.passKeyService.listUserCredentials(userId);
        return {
            credentials: credentials.map(c => ({
                id: c.id,
                deviceName: c.device_name,
                deviceType: c.device_type,
                lastUsedAt: c.last_used_at,
                createdAt: c.created_at
            }))
        };
    }

    async deletePassKey(userId: string, credentialId: string, req: express.Request) {
        await this.passKeyService.deleteCredential(credentialId, userId);

        await logActivity(
            userId,
            "auth",
            "passkey_deleted",
            "info",
            "User deleted a PassKey",
            req,
            getDeviceFingerprint(req)
        );

        return { success: true, message: "PassKey deleted successfully" };
    }

    async ldapSignin(username: string, password: string, req: express.Request) {
        if (!username || !password) {
            throw new AuthError("auth.errors.missingCredentials", 400);
        }

        try {
            const ldapUser = await this.ldapService.authenticate(username, password);
            let localUserId = await this.ldapService.findUserByLDAPUID(ldapUser.uid);

            if (!localUserId) {
                const newUser = await this.userRepo.createUser(ldapUser.uid, ldapUser.email, "", true);
                localUserId = newUser.id;
                await this.securityRepo.updateAuthProvider(localUserId, "ldap", ldapUser.uid);
            }

            await this.ldapService.syncUser(localUserId, ldapUser);

            const fingerprint = getDeviceFingerprint(req);
            const ipAddress = getClientIpAddress(req);
            const userAgent = String(req.headers["user-agent"] || "");

            const session = await lucia.createSession(localUserId, {
                ip_address: ipAddress,
                user_agent: userAgent,
                device_fingerprint: fingerprint
            });

            await logActivity(
                localUserId,
                "auth",
                "ldap_login",
                "info",
                "User logged in with LDAP",
                req,
                fingerprint
            );

            const user = await this.userRepo.getById(localUserId);
            if (!user) {
                throw new AuthError("auth.errors.userNotFound", 404);
            }

            return {
                success: true,
                sessionCookie: lucia.createSessionCookie(session.id),
                user: {
                    id: user.id,
                    username: user.username,
                    email: user.email,
                    is_verified: user.is_verified
                }
            };
        } catch (error: any) {
            await logActivity(
                null,
                "auth",
                "ldap_login_failed",
                "medium",
                `LDAP login failed: ${error.message}`,
                req,
                getDeviceFingerprint(req),
                { attempted_username: username }
            );
            throw new AuthError("auth.errors.invalidCredentials", 401);
        }
    }

    async getSSOProviders() {
        const providers = await this.ssoService.getEnabledProviders();
        return {
            providers: providers.map(p => ({
                id: p.id,
                name: p.name,
                type: p.type
            }))
        };
    }

    async initiateSSOLogin(providerName: string, state: string) {
        const authUrl = await this.ssoService.generateAuthorizationUrl(providerName, state);
        return { authUrl };
    }

    async handleSSOCallback(providerName: string, code: string, req: express.Request) {
        const codeStr = code || "";
        if (!codeStr) {
            throw new AuthError("auth.errors.missingCode", 400);
        }

        const tokens = await this.ssoService.exchangeCodeForTokens(providerName, codeStr);
        const userInfo = await this.ssoService.getUserInfo(providerName, tokens.access_token);

        const ssoProvider = await this.ssoService.getProviderByName(providerName);
        if (!ssoProvider) {
            throw new AuthError("auth.errors.providerNotFound", 404);
        }

        let localUserId = await this.ssoService.findUserByProviderAndExternalId(ssoProvider.id, userInfo.id);

        if (!localUserId) {
            const newUser = await this.userRepo.createUser(
                userInfo.username || userInfo.email!,
                userInfo.email!,
                "",
                true
            );
            localUserId = newUser.id;
            await this.securityRepo.updateAuthProvider(localUserId, "sso", userInfo.id);
        }

        await this.ssoService.linkUser(localUserId, ssoProvider.id, userInfo.id, userInfo, tokens);

        const fingerprint = getDeviceFingerprint(req);
        const ipAddress = getClientIpAddress(req);
        const userAgent = String(req.headers["user-agent"] || "");

        const session = await lucia.createSession(localUserId, {
            ip_address: ipAddress,
            user_agent: userAgent,
            device_fingerprint: fingerprint
        });

        await logActivity(
            localUserId,
            "auth",
            "sso_login",
            "info",
            `User logged in with SSO (${providerName})`,
            req,
            fingerprint
        );

        const frontendUrl = process.env.FRONTEND_URL || "http://localhost:5173";
        return {
            sessionCookie: lucia.createSessionCookie(session.id),
            redirectUrl: `${frontendUrl}/dashboard`
        };
    }

    async getDevices(userId: string) {
        const devices = await this.securityService.listDevices(userId);
        return {
            devices: devices.map((d: any) => ({
                id: d.id,
                name: d.device_name,
                fingerprint: d.device_fingerprint,
                ipAddress: d.ip_address,
                lastActiveAt: d.last_active_at,
                isTrusted: d.is_trusted,
                trustedAt: d.trusted_at,
                isBlocked: d.is_blocked,
                createdAt: d.created_at
            }))
        };
    }

    async trustDevice(userId: string, deviceId: string, req: express.Request) {
        await this.securityService.trustDevice(userId, deviceId);

        await logActivity(
            userId,
            "security",
            "device_trusted",
            "info",
            "User trusted a device",
            req,
            getDeviceFingerprint(req),
            { device_id: deviceId }
        );

        return { success: true, message: "Device trusted successfully" };
    }

    async revokeDevice(userId: string, deviceId: string, req: express.Request) {
        await this.securityService.revokeDevice(userId, deviceId);

        await logActivity(
            userId,
            "security",
            "device_revoked",
            "medium",
            "User revoked a device",
            req,
            getDeviceFingerprint(req),
            { device_id: deviceId }
        );

        return { success: true, message: "Device revoked successfully" };
    }

    async getDeviceNotifications(userId: string, includeRead: boolean) {
        const notifications = await this.securityService.getDeviceNotifications(userId, includeRead);
        return { notifications };
    }

    async acknowledgeNotification(userId: string, notificationId: string) {
        await this.securityService.acknowledgeNotification(userId, notificationId);
        return { success: true };
    }

    async changePasswordHandler(
        sessionId: string, oldPass: string, newPass: string, data?: {
            captchaT?: string,
            ip_address?: string,
            user_agent?: string,
            device_fingerprint?: string,
            req?: express.Request
        }
    ): Promise<AuthResult> {

        try {
            // Check if IP is blocked before anything else (for security and to prevent user enumeration)
            const isBlocked = await this.authRepo.isIpBlocked(data?.ip_address || '');
            if (isBlocked) {
                await logActivity(
                    null, 'auth', 'ip_blocked', 'critical',
                    `Blocked signup attempt from IP: ${data?.ip_address}`,
                    data?.req, 
                    data?.device_fingerprint, 
                    { attempted_session_id: sessionId }
                );

                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            const { session } = await lucia.validateSession(sessionId);
            if (!session) {
                return { 
                    success: false, 
                    message: "auth.errors.notAuthenticated" 
                };
            }
            
            const { oldPassword, newPassword } = this.changePasswordSchema.parse({
                oldPassword: oldPass,
                newPassword: newPass
            });

            const user = await this.userRepo.getById(session.userId);
            if (!user) {
                return {
                    success: false,
                    message: "auth.errors.notAuthenticated"
                };
            }
            
            const currentHash = user.password_hash;
            const validPassword = await scrypt.verify(currentHash, getPepperedPassword(oldPassword));

            if (!validPassword) {
                await logActivity(
                    user.id, 'auth', 'password_change_failed', 'medium', 
                    'Invalid old password during change attempt', 
                    data?.req, data?.device_fingerprint, 
                    { user_id: user.id }
                );
                return { success: false, message: "auth.errors.invalidOldPassword" };
            }

            const newPasswordHash = await scrypt.hash(getPepperedPassword(newPassword));
            await pool.query("UPDATE users SET password_hash = $1 WHERE id = $2", [newPasswordHash, user.id]);

            await logActivity(
                user.id, 'auth', 'password_complete', 'high', 
                'Password changed successfully', 
                data?.req, data?.device_fingerprint, 
                { user_id: user.id }
            );

            return { success: true, message: "Password updated successfully" };

        } catch (error: any) {
            if (error instanceof z.ZodError) {
                return { 
                    success: false, 
                    message: "auth.errors.invalidInput", 
                    details: error.issues 
                };
            }
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Password change error: ${error.message}`, 
                data?.req, data?.device_fingerprint, 
                { error: error.message, stack: error.stack, attempted_session_id: sessionId }
            );
            console.error(error);
            return { 
                success: false, 
                message: "auth.errors.internalServer" 
            };
        }
    }
}