import { BaseRepository } from './BaseRepository';

export interface UserRow {
    id: string;
    username: string;
    email: string;
    password_hash: string;
    avatar_url?: string;
    is_verified?: boolean
    created_at?: Date;
    updated_at?: Date;
}

export interface UserProfilesRow {
    user_id: string;
    first_name?: string;
    middle_name?: string;
    last_name?: string;
    nickname?: string;
    birthday?: Date;
    id_number?: string;
    passport_number?: string;
    mobile_phone?: string;
    address?: string;
    created_at?: Date;
    updated_at?: Date;
}

export class UserRepository extends BaseRepository {
    
    /**
     * Find user by ID
     * @param id 
     * @returns 
     */
    async findById(id: string): Promise<UserRow | null> {
        const res = await this.query('SELECT * FROM users WHERE id = $1', [id]);
        return res.rows[0] || null;
    }

    /**
     * Find user by email
     * @param email 
     * @returns 
     */
    async findByEmail(email: string): Promise<UserRow | null> {
        const res = await this.query('SELECT * FROM users WHERE email = $1', [email]);
        return res.rows[0] || null;
    }

    /**
     * Find user by username
     * @param username 
     * @returns 
     */
    async findByUsername(username: string): Promise<UserRow | null> {
        const res = await this.query('SELECT * FROM users WHERE username = $1', [username]);
        return res.rows[0] || null;
    }

    /**
     * Create new user
     * @param username 
     * @param email 
     * @param passwordHash 
     * @param isVerified 
     * @returns 
     */
    async createUser(username: string, email: string, passwordHash: string, isVerified: boolean = false): Promise<UserRow> {
        const res = await this.query(
            'INSERT INTO users (username, email, password_hash, is_verified) VALUES ($1, $2, $3, $4) RETURNING *',
            [username, email, passwordHash, isVerified]
        );
        return res.rows[0];
    }

    /**
     * Update username
     * @param userId 
     * @param newUsername 
     */
    async updateUsername(userId: string, newUsername: string): Promise<void> {
        await this.query('UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2', [newUsername, userId]);
    }

    /**
     * Update email
     * @param userId 
     * @param newEmail 
     */
    async updateEmail(userId: string, newEmail: string): Promise<void> {
        await this.query('UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2', [newEmail, userId]);
    }

    /**
     * Update password
     * @param userId 
     * @param newPasswordHash 
     */
    async updatePassword(userId: string, newPasswordHash: string): Promise<void> {
        await this.query('UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2', [newPasswordHash, userId]);
    }

    /**
     * Update verification status
     * @param userId 
     * @param isVerified 
     */
    async updateVerificationStatus(userId: string, isVerified: boolean): Promise<void> {
        await this.query('UPDATE users SET is_verified = $1, updated_at = NOW() WHERE id = $2', [isVerified, userId]);
    }

    /**
     * Update avatar URL
     * @param userId 
     * @param avatarUrl 
     */
    async updateAvatar(userId: string, avatarUrl: string): Promise<void> {
        await this.query('UPDATE users SET avatar_url = $1, updated_at = NOW() WHERE id = $2', [avatarUrl, userId]);
    }

    /**
     * Create user profile
     * @param userId 
     * @param profileData 
     * @returns 
     */
    async createUserProfile(userId: string, profileData: Partial<UserProfilesRow>): Promise<UserProfilesRow> {
        const res = await this.query(
            `INSERT INTO user_profiles (user_id, first_name, middle_name, last_name, nickname, birthday, id_number, passport_number, mobile_phone, address)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *`,
            [userId, profileData.first_name, profileData.middle_name, profileData.last_name, profileData.nickname, profileData.birthday, profileData.id_number, profileData.passport_number, profileData.mobile_phone, profileData.address]
        );
        return res.rows[0];
    }

    /**
     * Update user profile
     * @param userId 
     * @param profileData 
     * @returns 
     */
    async updateUserProfile(userId: string, profileData: Partial<UserProfilesRow>): Promise<UserProfilesRow> {
        const res = await this.query(
            `UPDATE user_profiles SET 
                first_name = COALESCE($2, first_name),
                middle_name = COALESCE($3, middle_name),
                last_name = COALESCE($4, last_name),
                nickname = COALESCE($5, nickname),
                birthday = COALESCE($6, birthday),
                id_number = COALESCE($7, id_number),
                passport_number = COALESCE($8, passport_number),
                mobile_phone = COALESCE($9, mobile_phone),
                address = COALESCE($10, address)
             WHERE user_id = $1 RETURNING *`,
            [userId, profileData.first_name, profileData.middle_name, profileData.last_name, profileData.nickname, profileData.birthday, profileData.id_number, profileData.passport_number, profileData.mobile_phone, profileData.address]
        );
        return res.rows[0];
    }
}