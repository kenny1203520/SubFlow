import { Router } from "express";
import { authLimiter } from "../middleware/rateLimit";
import { checkIpBlocked, verifyCaptcha, injectSecurityContext, requireAuth, authenticateSession } from "../middleware/auth";
import { AuthController } from "../controllers/AuthController";

const router = Router();
const authController = new AuthController();

// ===================== Authentication Routes =====================

/**
 * Get captcha config
 */
router.get(
    "/config", 
    checkIpBlocked, 
    (req, res) => authController.getCaptchaConfig(req, res)
);

/**
 * Signup
 */
router.post(
    "/signup", 
    checkIpBlocked, 
    authLimiter, 
    verifyCaptcha, 
    injectSecurityContext, 
    (req, res) => authController.signupHandler(req, res)
);

/**
 * Signin
 */
router.post(
    "/signin", 
    checkIpBlocked, 
    authLimiter, 
    verifyCaptcha, 
    injectSecurityContext, 
    (req, res) => authController.signinHandler(req, res)
);

/**
 * 2FA Signin
 */
router.post(
    "/signin/2fa", 
    checkIpBlocked, 
    authLimiter, 
    verifyCaptcha, 
    injectSecurityContext, 
    (req, res) => authController.signin2faHandler(req, res)
);

/**
 * Signout
 */
router.post(
    "/signout", 
    checkIpBlocked,
    injectSecurityContext, 
    (req, res) => authController.signoutHandler(req, res)
);

/**
 * Get user
 */
router.get(
    "/user", 
    checkIpBlocked, 
    injectSecurityContext, 
    (req, res) => authController.userHandler(req, res)
);

/**
 * Verify email
 */
router.get(
    "/verify-email/:token", 
    checkIpBlocked, 
    injectSecurityContext, 
    (req, res) => authController.verifyEmailHandler(req, res)
);

/**
 * Password reset
 */
router.post(
    "/password-reset", 
    checkIpBlocked, 
    authLimiter, 
    verifyCaptcha, 
    injectSecurityContext, 
    (req, res) => authController.passwordResetHandler(req, res)
);

/**
 * Password reset token
 */
router.post(
    "/password-reset/:token", 
    checkIpBlocked, 
    authLimiter, 
    verifyCaptcha, 
    injectSecurityContext, 
    (req, res) => authController.passwordResetTokenHandler(req, res)
);

/**
 * Change password
 */
router.post(
    "/change-password", 
    checkIpBlocked, 
    requireAuth, 
    authLimiter, 
    injectSecurityContext, 
    (req, res) => authController.changePasswordHandler(req, res)
);

// ===================== PassKey/WebAuthn Routes =====================

/**
 * PassKey Registration
 * Requires authentication
 */
router.post(
    '/passkey/register/options',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.generatePassKeyRegistrationOptions(req, res)
);

router.post(
    '/passkey/register/verify',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.verifyPassKeyRegistration(req, res)
);

/**
 * PassKey Authentication
 * Public routes
 */
router.post(
    '/passkey/authenticate/options',
    checkIpBlocked, 
    (req, res) => authController.generatePassKeyAuthenticationOptions(req, res)
);

router.post(
    '/passkey/authenticate/verify',
    checkIpBlocked, 
    (req, res) => authController.verifyPassKeyAuthentication(req, res)
);

/**
 * PassKey Management
 * Requires authentication
 */
router.get(
    '/passkey/list',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.listPassKeys(req, res)
);

router.delete(
    '/passkey/:id',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.deletePassKey(req, res)
);

// ===================== LDAP Routes =====================

/**
 * LDAP Authentication
 * Public route
 */
router.post(
    '/ldap/signin',
    checkIpBlocked,
    authLimiter,
    verifyCaptcha,
    injectSecurityContext,
    (req, res) => authController.ldapSignin(req, res)
);

// ===================== SSO Routes =====================

/**
 * Get available SSO providers
 * Public route
 */
router.get(
    '/sso/providers',
    checkIpBlocked, 
    (req, res) => authController.getSSOProviders(req, res)
);

/**
 * Initiate SSO login
 * Public route - redirects to provider
 */
router.get(
    '/sso/:provider',
    checkIpBlocked, 
    (req, res) => authController.initiateSSOLogin(req, res)
);

/**
 * Handle SSO callback
 * Public route - called by provider after authentication
 */
router.get(
    '/sso/callback/:provider',
    checkIpBlocked, 
    injectSecurityContext,
    (req, res) => authController.handleSSOCallback(req, res)
);

// ===================== Device Management Routes =====================

/**
 * Get user's devices
 * Requires authentication
 */
router.get(
    '/devices',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.getDevices(req, res)
);

/**
 * Trust a device
 * Requires authentication
 */
router.post(
    '/devices/:id/trust',
    checkIpBlocked, 
    authenticateSession,
    injectSecurityContext,
    (req, res) => authController.trustDevice(req, res)
);

/**
 * Revoke a device (logout and block)
 * Requires authentication
 */
router.post(
    '/devices/:id/revoke',
    checkIpBlocked, 
    authenticateSession,
    injectSecurityContext,
    (req, res) => authController.revokeDevice(req, res)
);

/**
 * Get device notifications
 * Requires authentication
 */
router.get(
    '/devices/notifications',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.getDeviceNotifications(req, res)
);

/**
 * Acknowledge device notification
 * Requires authentication
 */
router.post(
    '/devices/notifications/:id/acknowledge',
    checkIpBlocked, 
    authenticateSession,
    (req, res) => authController.acknowledgeNotification(req, res)
);

export default router;
