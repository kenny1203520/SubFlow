import { BaseRepository } from '../repositories/BaseRepository';
import crypto from 'crypto';
import axios from 'axios';

/**
 * SSOService handles OAuth2/OIDC Single Sign-On authentication
 * Supports providers like Google, GitHub, Microsoft, Okta, etc.
 * 
 * For production use with popular providers, consider using:
 * - Passport.js with provider-specific strategies
 * - next-auth / auth.js
 * - Or implement OAuth2 flow manually
 */

interface SSOProvider {
    id: string;
    name: string;
    type: 'oauth2' | 'saml' | 'oidc';
    enabled: boolean;
    client_id?: string;
    client_secret?: string;
    authorization_url?: string;
    token_url?: string;
    userinfo_url?: string;
    scope?: string;
    metadata_url?: string;
    config?: Record<string, any>;
}

interface SSOUserMapping {
    id: string;
    user_id: string;
    provider_id: string;
    external_id: string;
    external_email?: string;
    external_username?: string;
    profile_data?: Record<string, any>;
    last_login_at?: Date;
}

interface OAuthTokenResponse {
    access_token: string;
    token_type: string;
    expires_in?: number;
    refresh_token?: string;
    scope?: string;
    id_token?: string; // For OIDC
}

interface OAuthUserInfo {
    id: string;
    email?: string;
    name?: string;
    username?: string;
    avatar?: string;
    [key: string]: any;
}

export class SSOService extends BaseRepository {
    
    private readonly CALLBACK_URL_BASE = process.env.BACKEND_URL || 'http://localhost:3000';

    /**
     * Get all enabled SSO providers
     */
    async getEnabledProviders(): Promise<SSOProvider[]> {
        const res = await this.query(`
            SELECT * FROM sso_providers
            WHERE enabled = true
            ORDER BY name
        `);
        return res.rows;
    }

    /**
     * Get SSO provider by name
     */
    async getProviderByName(name: string): Promise<SSOProvider | null> {
        const res = await this.query(`
            SELECT * FROM sso_providers
            WHERE name = $1
        `, [name]);
        return res.rows[0] || null;
    }

    /**
     * Create or update SSO provider configuration
     */
    async upsertProvider(data: Partial<SSOProvider>): Promise<SSOProvider> {
        const id = data.id || crypto.randomUUID();
        
        // Encrypt client_secret before storing
        const encryptedSecret = data.client_secret 
            ? this.encrypt(data.client_secret)
            : null;

        const res = await this.query(`
            INSERT INTO sso_providers (
                id, name, type, enabled, client_id, client_secret,
                authorization_url, token_url, userinfo_url, scope,
                metadata_url, config
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (id) DO UPDATE SET
                name = EXCLUDED.name,
                type = EXCLUDED.type,
                enabled = EXCLUDED.enabled,
                client_id = EXCLUDED.client_id,
                client_secret = EXCLUDED.client_secret,
                authorization_url = EXCLUDED.authorization_url,
                token_url = EXCLUDED.token_url,
                userinfo_url = EXCLUDED.userinfo_url,
                scope = EXCLUDED.scope,
                metadata_url = EXCLUDED.metadata_url,
                config = EXCLUDED.config,
                updated_at = CURRENT_TIMESTAMP
            RETURNING *
        `, [
            id, data.name, data.type, data.enabled ?? true,
            data.client_id, encryptedSecret,
            data.authorization_url, data.token_url, data.userinfo_url,
            data.scope, data.metadata_url,
            data.config ? JSON.stringify(data.config) : null
        ]);

        return res.rows[0];
    }

    /**
     * Generate authorization URL for OAuth2 flow
     */
    async generateAuthorizationUrl(providerName: string, state?: string): Promise<string> {
        const provider = await this.getProviderByName(providerName);
        if (!provider || !provider.enabled) {
            throw new Error(`Provider ${providerName} not found or not enabled`);
        }

        if (!provider.authorization_url || !provider.client_id) {
            throw new Error(`Provider ${providerName} configuration incomplete`);
        }

        const callbackUrl = `${this.CALLBACK_URL_BASE}/auth/sso/callback/${providerName}`;
        const stateParam = state || crypto.randomBytes(16).toString('hex');
        
        const params = new URLSearchParams({
            client_id: provider.client_id,
            redirect_uri: callbackUrl,
            response_type: 'code',
            scope: provider.scope || 'openid email profile',
            state: stateParam
        });

        return `${provider.authorization_url}?${params.toString()}`;
    }

    /**
     * Exchange authorization code for tokens
     */
    async exchangeCodeForTokens(
        providerName: string,
        code: string,
        state?: string
    ): Promise<OAuthTokenResponse> {
        const provider = await this.getProviderByName(providerName);
        if (!provider) {
            throw new Error(`Provider ${providerName} not found`);
        }

        if (!provider.token_url || !provider.client_id || !provider.client_secret) {
            throw new Error(`Provider ${providerName} configuration incomplete`);
        }

        const callbackUrl = `${this.CALLBACK_URL_BASE}/auth/sso/callback/${providerName}`;
        const decryptedSecret = this.decrypt(provider.client_secret);

        try {
            const response = await axios.post<OAuthTokenResponse>(
                provider.token_url,
                new URLSearchParams({
                    grant_type: 'authorization_code',
                    code,
                    redirect_uri: callbackUrl,
                    client_id: provider.client_id,
                    client_secret: decryptedSecret
                }),
                {
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                        'Accept': 'application/json'
                    }
                }
            );

            return response.data;
        } catch (error: any) {
            console.error(`[SSO] Token exchange failed for ${providerName}:`, error.response?.data || error.message);
            throw new Error(`Failed to exchange code for tokens: ${error.message}`);
        }
    }

    /**
     * Get user information from provider
     */
    async getUserInfo(providerName: string, accessToken: string): Promise<OAuthUserInfo> {
        const provider = await this.getProviderByName(providerName);
        if (!provider || !provider.userinfo_url) {
            throw new Error(`Provider ${providerName} not found or userinfo_url not configured`);
        }

        try {
            const response = await axios.get(provider.userinfo_url, {
                headers: {
                    'Authorization': `Bearer ${accessToken}`,
                    'Accept': 'application/json'
                }
            });

            // Normalize user info based on provider
            return this.normalizeUserInfo(providerName, response.data);
        } catch (error: any) {
            console.error(`[SSO] Get user info failed for ${providerName}:`, error.response?.data || error.message);
            throw new Error(`Failed to get user info: ${error.message}`);
        }
    }

    /**
     * Normalize user info from different providers to a common format
     */
    private normalizeUserInfo(providerName: string, rawData: any): OAuthUserInfo {
        switch (providerName.toLowerCase()) {
            case 'google':
                return {
                    id: rawData.sub || rawData.id,
                    email: rawData.email,
                    name: rawData.name,
                    username: rawData.email?.split('@')[0],
                    avatar: rawData.picture,
                    email_verified: rawData.email_verified,
                    ...rawData
                };
            
            case 'github':
                return {
                    id: String(rawData.id),
                    email: rawData.email,
                    name: rawData.name,
                    username: rawData.login,
                    avatar: rawData.avatar_url,
                    ...rawData
                };
            
            case 'microsoft':
                return {
                    id: rawData.id || rawData.oid,
                    email: rawData.email || rawData.userPrincipalName,
                    name: rawData.displayName || rawData.name,
                    username: rawData.userPrincipalName?.split('@')[0],
                    avatar: null,
                    ...rawData
                };
            
            default:
                // Generic normalization
                return {
                    id: rawData.id || rawData.sub || rawData.user_id,
                    email: rawData.email,
                    name: rawData.name || rawData.displayName,
                    username: rawData.username || rawData.preferred_username || rawData.email?.split('@')[0],
                    avatar: rawData.avatar || rawData.picture || rawData.avatar_url,
                    ...rawData
                };
        }
    }

    /**
     * Link SSO account to local user
     */
    async linkUser(
        localUserId: string,
        providerId: string,
        externalId: string,
        userInfo: OAuthUserInfo,
        tokens?: OAuthTokenResponse
    ): Promise<SSOUserMapping> {
        const encryptedAccessToken = tokens?.access_token 
            ? this.encrypt(tokens.access_token)
            : null;
        
        const encryptedRefreshToken = tokens?.refresh_token
            ? this.encrypt(tokens.refresh_token)
            : null;

        const tokenExpiresAt = tokens?.expires_in
            ? new Date(Date.now() + tokens.expires_in * 1000)
            : null;

        const res = await this.query(`
            INSERT INTO sso_users (
                user_id, provider_id, external_id, external_email,
                external_username, access_token, refresh_token,
                token_expires_at, profile_data, last_login_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
            ON CONFLICT (provider_id, external_id) DO UPDATE SET
                user_id = EXCLUDED.user_id,
                external_email = EXCLUDED.external_email,
                external_username = EXCLUDED.external_username,
                access_token = EXCLUDED.access_token,
                refresh_token = EXCLUDED.refresh_token,
                token_expires_at = EXCLUDED.token_expires_at,
                profile_data = EXCLUDED.profile_data,
                last_login_at = CURRENT_TIMESTAMP,
                updated_at = CURRENT_TIMESTAMP
            RETURNING *
        `, [
            localUserId, providerId, externalId,
            userInfo.email, userInfo.username,
            encryptedAccessToken, encryptedRefreshToken,
            tokenExpiresAt, JSON.stringify(userInfo)
        ]);

        return res.rows[0];
    }

    /**
     * Find local user by SSO provider and external ID
     */
    async findUserByProviderAndExternalId(providerId: string, externalId: string): Promise<string | null> {
        const res = await this.query(`
            SELECT user_id FROM sso_users
            WHERE provider_id = $1 AND external_id = $2
        `, [providerId, externalId]);
        return res.rows[0]?.user_id || null;
    }

    /**
     * Get SSO mappings for a user
     */
    async getUserSSOAccounts(userId: string): Promise<SSOUserMapping[]> {
        const res = await this.query(`
            SELECT su.*, sp.name as provider_name
            FROM sso_users su
            JOIN sso_providers sp ON su.provider_id = sp.id
            WHERE su.user_id = $1
        `, [userId]);
        return res.rows;
    }

    /**
     * Unlink SSO account from user
     */
    async unlinkUser(userId: string, providerId: string): Promise<void> {
        await this.query(`
            DELETE FROM sso_users
            WHERE user_id = $1 AND provider_id = $2
        `, [userId, providerId]);
    }

    // ===================== Encryption Helpers =====================

    private encrypt(text: string): string {
        const key = this.getEncryptionKey();
        const iv = crypto.randomBytes(16);
        const cipher = crypto.createCipheriv('aes-256-cbc', key, iv);
        
        let encrypted = cipher.update(text, 'utf8', 'hex');
        encrypted += cipher.final('hex');
        
        return `${iv.toString('hex')}:${encrypted}`;
    }

    private decrypt(encryptedText: string): string {
        const key = this.getEncryptionKey();
        const [ivHex, encrypted] = encryptedText.split(':');
        const iv = Buffer.from(ivHex, 'hex');
        const decipher = crypto.createDecipheriv('aes-256-cbc', key, iv);
        
        let decrypted = decipher.update(encrypted, 'hex', 'utf8');
        decrypted += decipher.final('utf8');
        
        return decrypted;
    }

    private getEncryptionKey(): Buffer {
        const secret = process.env.ENCRYPTION_SECRET || process.env.AUTH_SECRET;
        if (!secret) {
            throw new Error('ENCRYPTION_SECRET or AUTH_SECRET not set');
        }
        return crypto.createHash('sha256').update(secret).digest();
    }

    // ===================== Provider Presets =====================

    /**
     * Initialize common SSO providers with default configurations
     */
    async initializeCommonProviders(): Promise<void> {
        const providers = [
            {
                name: 'google',
                type: 'oidc' as const,
                enabled: !!process.env.GOOGLE_CLIENT_ID,
                client_id: process.env.GOOGLE_CLIENT_ID,
                client_secret: process.env.GOOGLE_CLIENT_SECRET,
                authorization_url: 'https://accounts.google.com/o/oauth2/v2/auth',
                token_url: 'https://oauth2.googleapis.com/token',
                userinfo_url: 'https://www.googleapis.com/oauth2/v2/userinfo',
                scope: 'openid email profile'
            },
            {
                name: 'github',
                type: 'oauth2' as const,
                enabled: !!process.env.GITHUB_CLIENT_ID,
                client_id: process.env.GITHUB_CLIENT_ID,
                client_secret: process.env.GITHUB_CLIENT_SECRET,
                authorization_url: 'https://github.com/login/oauth/authorize',
                token_url: 'https://github.com/login/oauth/access_token',
                userinfo_url: 'https://api.github.com/user',
                scope: 'read:user user:email'
            },
            {
                name: 'microsoft',
                type: 'oidc' as const,
                enabled: !!process.env.MICROSOFT_CLIENT_ID,
                client_id: process.env.MICROSOFT_CLIENT_ID,
                client_secret: process.env.MICROSOFT_CLIENT_SECRET,
                authorization_url: 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize',
                token_url: 'https://login.microsoftonline.com/common/oauth2/v2.0/token',
                userinfo_url: 'https://graph.microsoft.com/v1.0/me',
                scope: 'openid email profile User.Read'
            }
        ];

        for (const provider of providers) {
            if (provider.enabled) {
                await this.upsertProvider(provider);
                // console.log(`[SSO] Initialized provider: ${provider.name}`);
            }
        }
    }
}
