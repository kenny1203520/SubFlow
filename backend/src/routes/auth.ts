import { Router } from "express";
import { lucia } from "../auth/lucia";
import { generateId } from "lucia";
import { Scrypt } from "oslo/password";
import { pool } from "../db";
import { MailService } from "../services/MailService";
import crypto from "crypto";

const router = Router();
const scrypt = new Scrypt();

router.post("/signup", async (req, res) => {
    const { username, email, password } = req.body;

    if (typeof username !== "string" || username.length < 3) {
        return res.status(400).send("Invalid username");
    }
    if (typeof password !== "string" || password.length < 6) {
        return res.status(400).send("Invalid password");
    }
    if (typeof email !== "string" || !email.includes("@")) {
        return res.status(400).send("Invalid email");
    }

    const passwordHash = await scrypt.hash(password);
    const userId = generateId(15);

    try {
        await pool.query(
            "INSERT INTO users (id, username, email, password_hash, is_verified) VALUES ($1, $2, $3, $4, $5)",
            [userId, username, email, passwordHash, false]
        );

        // Generate Verification Token
        const token = crypto.randomBytes(32).toString("hex");
        const tokenHash = crypto.createHash("sha256").update(token).digest("hex");
        const expiresAt = new Date(Date.now() + 1000 * 60 * 60 * 24); // 24 hours

        await pool.query(
            "INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at) VALUES ($1, $2, $3, $4)",
            [userId, email, tokenHash, expiresAt]
        );

        await MailService.sendVerificationEmail(email, token);

        const session = await lucia.createSession(userId, {});
        const sessionCookie = lucia.createSessionCookie(session.id);

        res.setHeader("Set-Cookie", sessionCookie.serialize());
        return res.status(201).json({ success: true, message: "Signup successful. Please verify your email." });
    } catch (error: any) {
        if (error.code === '23505') {
            return res.status(400).send("Username or email already exists");
        }
        console.error(error);
        return res.status(500).send("Unknown error");
    }
});

router.post("/signin", async (req, res) => {
    const { username, password } = req.body;

    if (typeof username !== "string" || typeof password !== "string") {
        return res.status(400).send("Invalid input");
    }

    try {
        const result = await pool.query("SELECT * FROM users WHERE username = $1", [username]);
        const user = result.rows[0];

        if (!user) {
            return res.status(400).send("Invalid username or password");
        }

        const validPassword = await scrypt.verify(user.password_hash, password);
        if (!validPassword) {
            return res.status(400).send("Invalid username or password");
        }

        const session = await lucia.createSession(user.id, {});
        const sessionCookie = lucia.createSessionCookie(session.id);

        res.setHeader("Set-Cookie", sessionCookie.serialize());
        return res.status(200).json({
            success: true,
            user: {
                id: user.id,
                username: user.username,
                email: user.email,
                avatar_url: user.avatar_url,
                is_verified: user.is_verified
            }
        });
    } catch (error) {
        console.error(error);
        return res.status(500).send("Internal Server Error");
    }
});

router.post("/signout", async (req, res) => {
    const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "");
    if (!sessionId) {
        return res.status(401).send("Not authenticated");
    }

    await lucia.invalidateSession(sessionId);

    const sessionCookie = lucia.createBlankSessionCookie();
    res.setHeader("Set-Cookie", sessionCookie.serialize());
    return res.status(200).send("Signed out");
});

router.get("/user", async (req, res) => {
    const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "");
    if (!sessionId) {
        return res.status(401).send("Not authenticated");
    }

    const { session, user } = await lucia.validateSession(sessionId);
    if (!session) {
        const sessionCookie = lucia.createBlankSessionCookie();
        res.setHeader("Set-Cookie", sessionCookie.serialize());
        return res.status(401).send("Not authenticated");
    }

    if (session && session.fresh) {
        const sessionCookie = lucia.createSessionCookie(session.id);
        res.setHeader("Set-Cookie", sessionCookie.serialize());
    }

    return res.status(200).json(user);
});

router.get("/verify-email/:token", async (req, res) => {
    const { token } = req.params;
    const tokenHash = crypto.createHash("sha256").update(token).digest("hex");

    try {
        const tokenRes = await pool.query(
            "SELECT * FROM email_verification_tokens WHERE token_hash = $1 AND expires_at > NOW()",
            [tokenHash]
        );
        const verificationToken = tokenRes.rows[0];

        if (!verificationToken) {
            return res.status(400).send("Invalid or expired verification token");
        }

        await pool.query("BEGIN");
        await pool.query("UPDATE users SET is_verified = TRUE WHERE id = $1", [verificationToken.user_id]);
        await pool.query("DELETE FROM email_verification_tokens WHERE id = $1", [verificationToken.id]);
        await pool.query("COMMIT");

        return res.status(200).json({ success: true, message: "Email verified successfully." });
    } catch (error) {
        await pool.query("ROLLBACK");
        console.error(error);
        return res.status(500).send("Server Error");
    }
});

router.post("/password-reset", async (req, res) => {
    const { email } = req.body;
    if (typeof email !== "string" || !email.includes("@")) {
        return res.status(400).send("Invalid email");
    }

    try {
        const userRes = await pool.query("SELECT id FROM users WHERE email = $1", [email]);
        const user = userRes.rows[0];
        if (!user) {
            return res.status(200).json({ success: true, message: "If account exists, email sent." });
        }

        const token = crypto.randomBytes(32).toString("hex");
        const tokenHash = crypto.createHash("sha256").update(token).digest("hex");
        const expiresAt = new Date(Date.now() + 1000 * 60 * 60); // 1 hour

        await pool.query(
            "INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
            [user.id, tokenHash, expiresAt]
        );

        await MailService.sendPasswordResetEmail(email, token);

        return res.status(200).json({ success: true, message: "Reset email sent." });
    } catch (error) {
        console.error(error);
        return res.status(500).send("Server Error");
    }
});

router.post("/password-reset/:token", async (req, res) => {
    const { token } = req.params;
    const { password } = req.body;

    if (typeof password !== "string" || password.length < 6) {
        return res.status(400).send("Invalid password");
    }

    const tokenHash = crypto.createHash("sha256").update(token).digest("hex");

    try {
        const tokenRes = await pool.query(
            "SELECT * FROM password_reset_tokens WHERE token_hash = $1 AND is_used = FALSE AND expires_at > NOW()",
            [tokenHash]
        );
        const resetToken = tokenRes.rows[0];

        if (!resetToken) {
            return res.status(400).send("Invalid or expired token");
        }

        const passwordHash = await scrypt.hash(password);

        await pool.query("BEGIN");
        await pool.query("UPDATE users SET password_hash = $1 WHERE id = $2", [passwordHash, resetToken.user_id]);
        await pool.query("UPDATE password_reset_tokens SET is_used = TRUE WHERE id = $1", [resetToken.id]);
        await pool.query("COMMIT");

        return res.status(200).json({ success: true, message: "Password updated." });
    } catch (error) {
        await pool.query("ROLLBACK");
        console.error(error);
        return res.status(500).send("Server Error");
    }
});

export default router;
