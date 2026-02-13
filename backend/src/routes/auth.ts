import { Router } from "express";
import { lucia } from "../auth/lucia";
import { generateId } from "lucia";
import { Scrypt } from "oslo/password";
import { pool } from "../db";

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
            "INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, $4)",
            [userId, username, email, passwordHash]
        );

        const session = await lucia.createSession(userId, {});
        const sessionCookie = lucia.createSessionCookie(session.id);

        res.setHeader("Set-Cookie", sessionCookie.serialize());
        return res.status(201).json({ success: true });
    } catch (error: any) {
        // Postgres error code for unique violation is 23505
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
        return res.status(200).json({ success: true, user: { id: user.id, username: user.username, email: user.email } });
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

export default router;
