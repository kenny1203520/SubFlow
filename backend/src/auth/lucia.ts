import { Lucia } from "lucia";
import { NodePostgresAdapter } from "@lucia-auth/adapter-postgresql";
import { pool } from "../db";

const adapter = new NodePostgresAdapter(pool, {
    user: "users",
    session: "sessions"
});

export const lucia = new Lucia(adapter, {
    sessionCookie: {
        attributes: {
            secure: process.env.NODE_ENV === "production"
        }
    },
    getUserAttributes: (attributes) => {
        return {
            username: attributes.username,
            email: attributes.email,
            avatar_url: attributes.avatar_url,
            is_verified: attributes.is_verified
        };
    },
    getSessionAttributes: (attributes) => {
        return {
            ip_address: attributes.ip_address,
            user_agent: attributes.user_agent,
            device_fingerprint: attributes.device_fingerprint
        };
    }
});

declare module "lucia" {
    interface Register {
        Lucia: typeof lucia;
        DatabaseUserAttributes: DatabaseUserAttributes;
        DatabaseSessionAttributes: DatabaseSessionAttributes;
    }
}

interface DatabaseUserAttributes {
    username: string;
    email: string;
    avatar_url: string;
    is_verified: boolean;
}

interface DatabaseSessionAttributes {
    ip_address: string;
    user_agent: string;
    device_fingerprint: string;
}
