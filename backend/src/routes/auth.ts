import { Router } from "express";
import { authLimiter } from "../middleware/rateLimit";
import { requireAuth, authenticateSession } from "../middleware/auth";
import { AuthController } from "../controllers/AuthController";

const router = Router();

// ===================== Authentication Routes =====================

// Get captcha config
router.get("/config", AuthController.getCaptchaConfig);

// Signup
router.post("/signup", authLimiter, AuthController.signupHandler);

// Signin
router.post("/signin", authLimiter, AuthController.signinHandler);

// 2FA Signin
router.post("/signin/2fa", authLimiter, AuthController.signin2faHandler);

// Signout
router.post("/signout", AuthController.signoutHandler);

// Get user
router.get("/user", AuthController.userHandler);

// Verify email
router.get("/verify-email/:token", AuthController.verifyEmailHandler);

// Password reset
router.post("/password-reset", authLimiter, AuthController.passwordResetHandler);

// Password reset token
router.post("/password-reset/:token", authLimiter, AuthController.passwordResetTokenHandler);

// Change password
router.post("/change-password", requireAuth, authLimiter, AuthController.changePasswordHandler);

// ===================== PassKey/WebAuthn Routes =====================

/**
 * PassKey Registration
 * Requires authentication
 */
router.post(
    '/passkey/register/options',
    authenticateSession,
    (req, res) => AuthController.generatePassKeyRegistrationOptions(req, res)
);

router.post(
    '/passkey/register/verify',
    authenticateSession,
    (req, res) => AuthController.verifyPassKeyRegistration(req, res)
);

/**
 * PassKey Authentication
 * Public routes
 */
router.post(
    '/passkey/authenticate/options',
    (req, res) => AuthController.generatePassKeyAuthenticationOptions(req, res)
);

router.post(
    '/passkey/authenticate/verify',
    (req, res) => AuthController.verifyPassKeyAuthentication(req, res)
);

/**
 * PassKey Management
 * Requires authentication
 */
router.get(
    '/passkey/list',
    authenticateSession,
    (req, res) => AuthController.listPassKeys(req, res)
);

router.delete(
    '/passkey/:id',
    authenticateSession,
    (req, res) => AuthController.deletePassKey(req, res)
);

// ===================== LDAP Routes =====================

/**
 * LDAP Authentication
 * Public route
 */
router.post(
    '/ldap/signin',
    (req, res) => AuthController.ldapSignin(req, res)
);

// ===================== SSO Routes =====================

/**
 * Get available SSO providers
 * Public route
 */
router.get(
    '/sso/providers',
    (req, res) => AuthController.getSSOProviders(req, res)
);

/**
 * Initiate SSO login
 * Public route - redirects to provider
 */
router.get(
    '/sso/:provider',
    (req, res) => AuthController.initiateSSOLogin(req, res)
);

/**
 * Handle SSO callback
 * Public route - called by provider after authentication
 */
router.get(
    '/sso/callback/:provider',
    (req, res) => AuthController.handleSSOCallback(req, res)
);

// ===================== Device Management Routes =====================

/**
 * Get user's devices
 * Requires authentication
 */
router.get(
    '/devices',
    authenticateSession,
    (req, res) => AuthController.getDevices(req, res)
);

/**
 * Trust a device
 * Requires authentication
 */
router.post(
    '/devices/:id/trust',
    authenticateSession,
    (req, res) => AuthController.trustDevice(req, res)
);

/**
 * Revoke a device (logout and block)
 * Requires authentication
 */
router.post(
    '/devices/:id/revoke',
    authenticateSession,
    (req, res) => AuthController.revokeDevice(req, res)
);

/**
 * Get device notifications
 * Requires authentication
 */
router.get(
    '/devices/notifications',
    authenticateSession,
    (req, res) => AuthController.getDeviceNotifications(req, res)
);

/**
 * Acknowledge device notification
 * Requires authentication
 */
router.post(
    '/devices/notifications/:id/acknowledge',
    authenticateSession,
    (req, res) => AuthController.acknowledgeNotification(req, res)
);

export default router;
