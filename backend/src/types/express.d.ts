// Type definitions for extended Express Request
import { Request } from 'express';

// Extend Express Request to include user and session
declare global {
    namespace Express {
        interface Request {
            user?: {
                id: string;
                username: string;
                email: string;
                is_verified: boolean;
                created_at: Date;
            };
            session?: {
                oauthState?: string;
                [key: string]: any;
            };
        }
    }
}

export {};
