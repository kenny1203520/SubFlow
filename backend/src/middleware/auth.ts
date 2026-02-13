import { Request, Response, NextFunction } from "express";
import { lucia } from "../auth/lucia";

export const verifySession = async (req: Request, res: Response, next: NextFunction) => {
    const sessionId = lucia.readSessionCookie(req.headers.cookie ?? "");
    if (!sessionId) {
        res.locals.user = null;
        res.locals.session = null;
        return next();
    }

    const { session, user } = await lucia.validateSession(sessionId);
    if (session && session.fresh) {
        res.setHeader("Set-Cookie", lucia.createSessionCookie(session.id).serialize());
    }
    if (!session) {
        res.setHeader("Set-Cookie", lucia.createBlankSessionCookie().serialize());
    }
    res.locals.user = user;
    res.locals.session = session;
    return next();
};

export const requireAuth = (req: Request, res: Response, next: NextFunction) => {
    if (!res.locals.session) {
        return res.status(401).json({ error: "Unauthorized" });
    }
    next();
};
