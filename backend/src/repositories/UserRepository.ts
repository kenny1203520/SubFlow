import { BaseRepository } from './BaseRepository';

export interface UserRow {
    id: string;
    username: string;
    email: string;
    password_hash: string;
    avatar_url?: string;
    created_at: Date;
}

export class UserRepository extends BaseRepository {
    async findById(id: string): Promise<UserRow | null> {
        const res = await this.query('SELECT * FROM users WHERE id = $1', [id]);
        return res.rows[0] || null;
    }

    async findByEmail(email: string): Promise<UserRow | null> {
        const res = await this.query('SELECT * FROM users WHERE email = $1', [email]);
        return res.rows[0] || null;
    }
}
