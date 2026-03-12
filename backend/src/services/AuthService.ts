import { lucia } from "../auth/lucia";
import { Scrypt } from "oslo/password";
import { pool } from "../db";
import { MailService } from "../services/MailService";
import crypto from "crypto";
import { z } from "zod";
import { SystemSettingsService } from "../services/SystemSettingsService";
import { verify } from "otplib";
import { logActivity } from "../utils/audit";
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

export interface SecurityContext {
    ipAddress: string;
    userAgent: string;
    deviceFingerprint: string;
}

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
    });

    public signinSchema = z.object({
        username: z.string().max(255).regex(/^[a-zA-Z0-9_-]+$/, "auth.errors.usernameFormat"),
        password: z.string().max(255),
    });

    public signin2FASchema = z.object({
        userId: z.uuid("auth.errors.invalidUserId"),
        code: z.string().min(6).max(8).regex(/^[0-9]+$/, "auth.errors.invalidTwoFactorCode"),
    });

    public resetSchema = z.object({
        password: z.string().min(8).max(255).regex(/[A-Z]/, "auth.errors.passwordUppercase").regex(/[0-9]/, "auth.errors.passwordNumber").regex(/[^A-Za-z0-9]/, "auth.errors.passwordSymbol"),
    });

    public changePasswordSchema = z.object({
        oldPassword: z.string().max(255),
        newPassword: z.string().min(8).max(255).regex(/[A-Z]/, "auth.errors.passwordUppercase").regex(/[0-9]/, "auth.errors.passwordNumber").regex(/[^A-Za-z0-9]/, "auth.errors.passwordSymbol"),
    });

    /**     
     * Signup handler to create a new user, send verification email, and auto-login
     * With device tracking and risk-based notifications
     * @param username 
     * @param email 
     * @param password 
     * @param context 
     * @returns 
     */
    async signupHandler(
        username: string,
        email: string,
        password: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        try {
            // Step 1: Validate input
            const { username: validUsername, email: validEmail, password: validPassword } = 
                this.signupSchema.parse({ username, email, password });

            // Step 2: Create user with hashed password
            const passwordHash = await scrypt.hash(getPepperedPassword(validPassword));
            const user = await this.userRepo.createUser(validUsername, validEmail, passwordHash, false);

            // Step 3: Create email verification token
            const token = crypto.randomBytes(32).toString("hex");
            const tokenHash = crypto.createHash("sha256").update(token).digest("hex");
            const expiresAt = new Date(Date.now() + 1000 * 60 * 60 * 24); // 24 hours

            await this.authRepo.createEmailVerificationToken(user.id, validEmail, tokenHash, expiresAt);
            
            // Step 4: Initialize User Security (with local auth provider)
            await this.securityRepo.createSecuritySettings(user.id);
            await this.securityRepo.updateAuthProvider(user.id, 'local', '');
            
            // Step 5: Assign Guest Role
            const guestRole = await this.rbacRepo.getSystemRoleByName("Guest");
            if (guestRole) {
                await this.rbacRepo.assignRoleToUser(user.id, guestRole.id);
            }

            // Step 6: Send verification email
            await MailService.sendVerificationEmail(validEmail, token);

            // Step 7: Create session (auto-login after signup)
            const session = await lucia.createSession(user.id, {
                ip_address: context.ipAddress,
                user_agent: context.userAgent,
                device_fingerprint: context.deviceFingerprint
            });

            // Step 8: Register or update device information
            const { device, isNewDevice } = await this.securityService.registerOrUpdateDevice(
                user.id,
                {
                    fingerprint: context.deviceFingerprint,
                    userAgent: context.userAgent,
                    ipAddress: context.ipAddress
                }
            );

            // Step 9: Create login history record for signup event
            await this.authRepo.createLoginHistory(
                user.id, 
                session.id, 
                new Date(),
                context.ipAddress, 
                context.userAgent,
                context.deviceFingerprint,
                device.id, 
                'success'
            );
            
            // Step 10: Send new device login notification (if applicable)
            if (isNewDevice) {
                await this.securityService.notifyNewDeviceLogin(user.id, device.id, {
                    deviceName: device.device_name || 'Unknown Device',
                    ipAddress: context.ipAddress,
                    location: device.location
                });
            }

            return {
                success: true,
                user: {
                    id: user.id,
                    username: validUsername,
                    email: validEmail,
                    is_verified: false
                },
                sessionCookie: lucia.createSessionCookie(session.id),
                message: "auth.signup.success"
            };
        } catch (error: any) {
            console.error("Signup error:", error);
            if (error instanceof z.ZodError) {
                throw new AuthError("auth.signup.validationFailed", 400, error.issues);
            }
            throw new AuthError("auth.signup.failed", 500);
        }
    }

    /**
     * Signin handler to authenticate user, enforce security policies, and create session
     * @param username 
     * @param password 
     * @param context 
     * @returns 
     */
    async signinHandler(
        username: string,
        password: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        try {
            // Step 1: Validate input
            const { username: validUsername, password: validPasswordInput } = 
                this.signinSchema.parse({ username, password });

            // Step 2: Query user by username (case-insensitive)
            const user = await this.userRepo.getByUsername(validUsername);
            if (!user) {
                // 記錄但返回統一錯誤（不暴露用戶存在性）
                await logActivity(
                    null, 'auth', 'login_failed_nonexistent', 'medium',
                    `Login attempt for non-existent user`,
                    undefined, 
                    context.deviceFingerprint,
                    { attempted_username: validUsername }
                );
                return { 
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            // 確保安全設定存在
            const securitySettings = await this.securityRepo.getSecuritySettings(user.id);
            if (!securitySettings) {
                await this.securityRepo.createSecuritySettings(user.id);
            }

            // Step 3: 檢查帳戶狀態（BEFORE 密碼驗證）
            const accountStatus = await this.getAccountSecurityStatus(user.id);
            
            if (accountStatus.isBlocked || accountStatus.isSuspended) {
                // 內部記錄詳細信息
                await logActivity(
                    user.id, 'auth', 'login_blocked_or_suspended', 'critical',
                    `Login blocked - Status: ${accountStatus.isBlocked ? 'BLOCKED' : 'SUSPENDED'}`,
                    undefined, 
                    context.deviceFingerprint,
                    {
                        user_id: user.id,
                        is_blocked: accountStatus.isBlocked,
                        is_suspended: accountStatus.isSuspended,
                        suspended_until: accountStatus.suspendedUntil
                    }
                );
                
                // 對外返回統一的模糊消息（與用戶不存在相同）
                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            // Step 4: 檢查認證提供者
            const authProvider = await this.securityRepo.getAuthProvider(user.id);
            if (authProvider && authProvider.auth_provider !== 'local') {
                await logActivity(
                    user.id, 'auth', 'login_wrong_provider', 'medium',
                    `Login attempted with local provider on ${authProvider.auth_provider} account`,
                    undefined, 
                    context.deviceFingerprint,
                    { attempted_username: validUsername }
                );
                return {
                    success: false,
                    message: `auth.errors.use${authProvider.auth_provider}`
                };
            }

            // Step 5: 計算登入風險分數
            const loginRiskScore = await this.calculateLoginRiskScore(user.id, context);

            // Step 6: 驗證密碼
            const isValidPassword = await scrypt.verify(user.password_hash, getPepperedPassword(validPasswordInput));
            
            if (!isValidPassword) {
                await this.handleFailedLoginAttempt(user.id, validUsername, context);
                return {
                    success: false,
                    message: "auth.errors.invalidCredentials"
                };
            }

            // Step 7: 重設失敗計數
            await this.securityRepo.resetFailedLoginAttempts(user.id);

            // Step 8: 檢查 2FA
            const shouldRequire2FA = securitySettings?.two_factor_enabled || loginRiskScore >= 40;
            
            if (shouldRequire2FA && securitySettings?.two_factor_enabled) {
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
                    undefined, 
                    context.deviceFingerprint,
                    { username: validUsername, risk_score: loginRiskScore }
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

            // Step 9: 建立 session（無需 2FA）
            return await this.createSessionAndNotify(user, context, loginRiskScore, validUsername);

        } catch (error: any) {
            console.error("Signin error:", error);
            if (error instanceof AuthError) throw error;
            return {
                success: false,
                message: "auth.errors.invalidCredentials"
            };
        }
    }

    /**
     * 私有方法：建立 session 和通知
     */
    private async createSessionAndNotify(
        user: any,
        context: SecurityContext,
        loginRiskScore: number,
        validUsername: string
    ): Promise<AuthResult> {
        // 註冊或更新設備
        const { device, isNewDevice } = await this.securityService.registerOrUpdateDevice(
            user.id,
            {
                fingerprint: context.deviceFingerprint,
                userAgent: context.userAgent,
                ipAddress: context.ipAddress
            }
        );

        // 建立 session
        const session = await lucia.createSession(user.id, {
            ip_address: context.ipAddress,
            user_agent: context.userAgent,
            device_fingerprint: context.deviceFingerprint
        });

        // 建立登入歷史
        await this.authRepo.createLoginHistory(
            user.id, 
            session.id, 
            new Date(),
            context.ipAddress,
            context.userAgent,
            context.deviceFingerprint,
            device.id, 
            'success'
        );

        // 審計日誌
        await logActivity(
            user.id, 'auth', 'login', 'info', 
            'User logged in', 
            undefined, 
            context.deviceFingerprint, 
            { 
                username: validUsername, 
                session_id: session.id, 
                device_id: device.id, 
                is_new_device: isNewDevice, 
                risk_score: loginRiskScore 
            }
        );

        // 發送通知
        if (isNewDevice) {
            await this.securityService.notifyNewDeviceLogin(user.id, device.id, {
                deviceName: device.device_name || 'Unknown Device',
                ipAddress: context.ipAddress,
                location: device.location
            });
        } else if (loginRiskScore >= 30) {
            // 高風險登入（已知設備）
            const userAgentInfo = parseUserAgent(context.userAgent);
            await MailService.sendSuspiciousActivityAlert(
                user.email,
                {
                    activityType: 'High Risk Login Detected',
                    details: `Login from ${userAgentInfo.browser} on ${userAgentInfo.os} with risk score ${loginRiskScore}`,
                    ipAddress: context.ipAddress,
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
            sessionCookie: lucia.createSessionCookie(session.id)
        };
    }

    /**
     * Get account security status including block and suspension state, with automatic expiration handling
     */
    private async getAccountSecurityStatus(userId: string) {
        const security = await this.securityRepo.getSecuritySettings(userId);
        
        if (!security) {
            return {
                isBlocked: false,
                isSuspended: false,
                suspendedUntil: null
            };
        }

        let isSuspended = false;
        let suspendedUntil = security.suspended_until;

        // 檢查 suspension 是否已過期
        if (security.is_suspended && suspendedUntil) {
            if (new Date(suspendedUntil) <= new Date()) {
                // 自動清除過期的 suspension
                await this.securityRepo.unSuspendUser(userId);
                isSuspended = false;
                suspendedUntil = undefined;
            } else {
                isSuspended = true;
            }
        }

        return {
            isBlocked: security.is_blocked,
            isSuspended: isSuspended,
            suspendedUntil: suspendedUntil
        };
    }

    /**
     * Failed login attempt handler to increment counters, apply lockout if necessary, and log activity
     */
    private async handleFailedLoginAttempt(
        userId: string,
        attemptedUsername: string,
        context: SecurityContext
    ) {
        const lockoutCfg = await SystemSettingsService.getSetting('security.auth_lockout');
        const maxAttempts = lockoutCfg?.maxFailedAttempts ?? 5;
        const lockoutDurationMins = lockoutCfg?.lockoutDurationMins ?? 720;

        await this.securityRepo.incrementFailedLoginAttempts(userId);
        const failedCount = await this.securityRepo.getFailedLoginAttempts(userId);

        if (failedCount >= maxAttempts) {
            const suspendUntil = new Date(Date.now() + lockoutDurationMins * 60 * 1000);
            await this.securityRepo.suspendUser(userId, suspendUntil);

            await logActivity(
                userId, 'auth', 'account_lockout', 'critical',
                `Account auto-locked after ${failedCount} failed attempts`,
                undefined, 
                context.deviceFingerprint,
                { attempted_username: attemptedUsername, failed_attempts: failedCount }
            );
        } else {
            await logActivity(
                userId, 'auth', 'login_failed', 'medium',
                `Failed login attempt (${failedCount}/${maxAttempts})`,
                undefined, 
                context.deviceFingerprint,
                { attempted_username: attemptedUsername, failed_attempts: failedCount }
            );
        }
    }

    /**
     * Count login risk score based on device familiarity, location, and recent activity
     */
    private async calculateLoginRiskScore(userId: string, context: SecurityContext): Promise<number> {
        const existingDevices = await this.securityRepo.getDevices(userId);
        const previousDevice = existingDevices.find(d => d.device_fingerprint === context.deviceFingerprint);
        const isNewDevice = !previousDevice;
        const isNewLocation = !existingDevices.some(d => 
            this.isSameLocation(d.ip_address ?? null, context.ipAddress)
        );

        const user = await this.userRepo.getById(userId);
        const accountAgeInDays = user?.created_at 
            ? Math.floor((Date.now() - user.created_at.getTime()) / (1000 * 60 * 60 * 24))
            : 0;

        const security = await this.securityRepo.getSecuritySettings(userId);
        const failedAttempts = security?.failed_login_attempts || 0;
        const timeSinceLastLogin = this.getTimeSinceLastLogin(existingDevices);

        return calculateLoginRiskScore({
            isNewDevice,
            isNewLocation,
            fingerprintSimilarity: previousDevice ? 1.0 : 0.0,
            failedAttemptsRecently: failedAttempts,
            accountAge: accountAgeInDays,
            timeSinceLastLogin: timeSinceLastLogin
        });
    }

    /**
     * Check if two IP addresses are from the same location (basic check based on first 3 octets)
     */
    private isSameLocation(ip1: string | null, ip2: string): boolean {
        if (!ip1) return false;
        return ip1.split('.').slice(0, 3).join('.') === ip2.split('.').slice(0, 3).join('.');
    }

    /**
     * Get hours since last login based on device activity
     */
    private getTimeSinceLastLogin(devices: any[]): number {
        const lastDevice = devices
            .filter(d => d.last_active_at)
            .sort((a, b) => 
                new Date(b.last_active_at!).getTime() - new Date(a.last_active_at!).getTime()
            )[0];
        
        return lastDevice && lastDevice.last_active_at
            ? Math.floor((Date.now() - new Date(lastDevice.last_active_at).getTime()) / (1000 * 60 * 60))
            : 99999;
    }

    async signin2faHandler(
        uId: string,
        co: string,
        context: SecurityContext
    ): Promise<AuthResult> {

        try {
            const { userId, code } = this.signin2FASchema.parse({
                userId: uId,
                code: co
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

            const fingerprint = context.deviceFingerprint;
            const ipAddress = context.ipAddress;
            const userAgent = context.userAgent;
            
            if (!isValid) {
                await logActivity(
                    userId, 'auth', '2fa_failed', 'high',
                    'Invalid 2FA code', 
                    undefined, fingerprint, 
                    { user_id: userId, used_backup_code: code.length === 8 }
                );
                await logActivity(
                    userId,
                    'auth',
                    '2fa_failed',
                    'high',
                    'Invalid 2FA code',
                    undefined,
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
                undefined, fingerprint, 
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
                undefined, context.deviceFingerprint, 
                { error: error.message, stack: error.stack, attempted_user_id: uId }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    /**
     * Signout handler to invalidate session and log activity.
     */
    async signoutHandler(
        sessionId: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        if (!sessionId) {
            return {
                success: false,
                message: "auth.errors.notAuthenticated"
            }
        }

        const ipAddress = context.ipAddress;
        const userAgent = context.userAgent;
        const fingerprint = context.deviceFingerprint;

        const { session } = await lucia.validateSession(sessionId);
        if (session) {
            await logActivity(
                session.userId, 'auth', 'signout', 'low', 
                'User signed out', 
                undefined, fingerprint, 
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

    /**
     * User handler to return current user info, system roles, and permissions based on session.
     * @param sessionId 
     * @param context 
     * @returns 
     */
    async userHandler(
        sessionId: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        
        try {
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
                undefined, context.deviceFingerprint,
                { error: error.message, stack: error.stack, attempted_session_id: sessionId }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    /**
     * Email verification handler to validate token, update user verification status, and promote role if necessary
     * @param token
     * @param context
     * @returns
    */
    async verifyEmailHandler(
        token: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        
        try {
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
                    undefined, context.deviceFingerprint, 
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
                    undefined, context.deviceFingerprint, 
                    { error: error.message, stack: error.stack }
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
                undefined, context.deviceFingerprint, 
                { error: error.message, stack: error.stack }
            );
            console.error(error);
            return {
                success: false,
                message: "auth.errors.internalServer"
            };
        }
    }

    /**
     * Password reset request handler to validate email, create reset token, send email, and log activity
     * @param eMail 
     * @param context 
     * @returns 
     */
    async passwordResetHandler(
        eMail: string,
        context: SecurityContext
    ): Promise<AuthResult> {
        
        if (typeof eMail !== "string" || !eMail.includes("@")) {
            return {
                success: false,
                message: "auth.errors.invalidEmail"
            };
        }
        const email = z.email().parse(eMail).toLowerCase(); // Validate email format

        try {
            const user = await this.userRepo.getByEmail(email);
            
            // Always return success to prevent enumeration
            if (!user) {
                await logActivity(
                    null, 'auth', 'reset_request_unknown', 'low', 
                    `Reset requested for non-existent: ${email}`, 
                    undefined, context.deviceFingerprint, 
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
                undefined, context.deviceFingerprint, 
                { email: user.email }
            );

            return { success: true, message: "If account exists, email sent." };
        } catch (error: any) {
            await logActivity(
                null, 'auth', 'system_error', 'high', 
                `Password reset request error: ${error.message}`, 
                undefined, context.deviceFingerprint, 
                { error: error.message, stack: error.stack, attempted_email: email }
            );
            console.error(error);
            return { success: false, message: "auth.errors.internalServer" };
        }
    }

    async passwordResetTokenHandler(
        token: string,
        pass: string,
        context: SecurityContext
    ): Promise<AuthResult> {

        try {
            const { password } = this.resetSchema.parse({
                password: pass
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
                undefined, context.deviceFingerprint, 
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
                undefined, context.deviceFingerprint, 
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
        context: SecurityContext
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
                undefined,
                context.deviceFingerprint
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
                undefined,
                context.deviceFingerprint
            );
            throw new AuthError(error.message || "auth.errors.registrationFailed", 400);
        }
    }

    async generatePassKeyAuthenticationOptions(userId?: string) {
        return this.passKeyService.generateAuthenticationOptions(userId);
    }

    async verifyPassKeyAuthentication(assertion: any, context: SecurityContext) {
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

            const fingerprint = context.deviceFingerprint;
            const ipAddress = context.ipAddress;
            const userAgent = context.userAgent;

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
                undefined,
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
                undefined,
                context.deviceFingerprint
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

    async deletePassKey(userId: string, credentialId: string, context: SecurityContext) {
        await this.passKeyService.deleteCredential(credentialId, userId);

        await logActivity(
            userId,
            "auth",
            "passkey_deleted",
            "info",
            "User deleted a PassKey",
            undefined,
            context.deviceFingerprint
        );

        return { success: true, message: "PassKey deleted successfully" };
    }

    async ldapSignin(username: string, password: string, context: SecurityContext) {
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

            const fingerprint = context.deviceFingerprint;
            const ipAddress = context.ipAddress;
            const userAgent = context.userAgent;

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
                undefined,
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
                undefined,
                context.deviceFingerprint,
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

    async handleSSOCallback(providerName: string, code: string, context: SecurityContext) {
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

        const fingerprint = context.deviceFingerprint;
        const ipAddress = context.ipAddress;
        const userAgent = context.userAgent;

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
            undefined,
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

    async trustDevice(userId: string, deviceId: string, context: SecurityContext) {
        await this.securityService.trustDevice(userId, deviceId);

        await logActivity(
            userId,
            "security",
            "device_trusted",
            "info",
            "User trusted a device",
            undefined,
            context.deviceFingerprint,
            { device_id: deviceId }
        );

        return { success: true, message: "Device trusted successfully" };
    }

    async revokeDevice(userId: string, deviceId: string, context: SecurityContext) {
        await this.securityService.revokeDevice(userId, deviceId);

        await logActivity(
            userId,
            "security",
            "device_revoked",
            "medium",
            "User revoked a device",
            undefined,
            context.deviceFingerprint,
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
        sessionId: string,
        oldPass: string,
        newPass: string,
        context: SecurityContext
    ): Promise<AuthResult> {

        try {
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
                    undefined, context.deviceFingerprint, 
                    { user_id: user.id }
                );
                return { success: false, message: "auth.errors.invalidOldPassword" };
            }

            const newPasswordHash = await scrypt.hash(getPepperedPassword(newPassword));
            await this.userRepo.updatePassword(user.id, newPasswordHash);

            await logActivity(
                user.id, 'auth', 'password_complete', 'high', 
                'Password changed successfully', 
                undefined, context.deviceFingerprint, 
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
                undefined, context.deviceFingerprint, 
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