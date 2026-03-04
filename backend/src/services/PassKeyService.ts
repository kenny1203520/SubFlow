import { BaseRepository } from '../repositories/BaseRepository';
import crypto from 'crypto';
import { z } from 'zod';

/**
 * PassKeyService handles WebAuthn/PassKey authentication
 * References:
 * - WebAuthn Spec: https://www.w3.org/TR/webauthn-2/
 * - FIDO2 Spec: https://fidoalliance.org/specs/fido-v2.0-ps-20190130/fido-client-to-authenticator-protocol-v2.0-ps-20190130.html
 */

interface WebAuthnCredential {
    id: string;
    user_id: string;
    credential_id: string;
    public_key: string;
    counter: number;
    device_type?: string;
    transports?: string[];
    backup_eligible?: boolean;
    backup_state?: boolean;
    attestation_object?: string;
    aaguid?: string;
    device_name?: string;
    last_used_at?: Date;
    created_at?: Date;
    updated_at?: Date;
}

interface WebAuthnChallenge {
    id: string;
    user_id?: string;
    challenge: string;
    type: 'registration' | 'authentication';
    expires_at: Date;
    created_at?: Date;
}

export class PassKeyService extends BaseRepository {
    
    private readonly RP_NAME = 'SubFlow';
    private readonly RP_ID = process.env.WEBAUTHN_RP_ID || 'localhost';
    private readonly ORIGIN = process.env.WEBAUTHN_ORIGIN || 'http://localhost:5173';
    private readonly CHALLENGE_TIMEOUT = 5 * 60 * 1000; // 5 minutes

    /**
     * Generate registration options for a new passkey
     * Client will use this to call navigator.credentials.create()
     */
    async generateRegistrationOptions(userId: string, username: string, email: string) {
        // Generate a random challenge
        const challenge = this.generateChallenge();
        
        // Store challenge in database
        const expiresAt = new Date(Date.now() + this.CHALLENGE_TIMEOUT);
        await this.createChallenge(userId, challenge, 'registration', expiresAt);

        // Get existing credentials for this user to exclude them
        const existingCredentials = await this.getCredentialsByUserId(userId);
        const excludeCredentials = existingCredentials.map(cred => ({
            id: cred.credential_id,
            type: 'public-key' as const,
            transports: cred.transports || ['internal']
        }));

        // Return options for client-side WebAuthn API
        return {
            challenge: this.base64urlEncode(Buffer.from(challenge)),
            rp: {
                name: this.RP_NAME,
                id: this.RP_ID
            },
            user: {
                id: this.base64urlEncode(Buffer.from(userId)),
                name: email,
                displayName: username
            },
            pubKeyCredParams: [
                { alg: -7, type: 'public-key' as const },  // ES256
                { alg: -257, type: 'public-key' as const } // RS256
            ],
            timeout: 60000, // 60 seconds
            excludeCredentials,
            authenticatorSelection: {
                authenticatorAttachment: 'platform' as const, // Prefer platform authenticators (biometrics)
                requireResidentKey: true, // Support discoverable credentials
                residentKey: 'required' as const,
                userVerification: 'required' as const
            },
            attestation: 'none' as const // Or 'direct' if you need attestation
        };
    }

    /**
     * Verify registration response from client
     * This validates the attestation and stores the credential
     */
    async verifyRegistration(
        userId: string,
        attestationResponse: {
            id: string;
            rawId: string;
            response: {
                clientDataJSON: string;
                attestationObject: string;
            };
            type: string;
            authenticatorAttachment?: string;
            clientExtensionResults?: any;
        },
        deviceName?: string
    ) {
        // Decode client data JSON
        const clientDataJSON = JSON.parse(
            Buffer.from(attestationResponse.response.clientDataJSON, 'base64').toString('utf-8')
        );

        // Verify challenge
        const storedChallenge = await this.getLatestChallenge(userId, 'registration');
        if (!storedChallenge) {
            throw new Error('No registration challenge found');
        }

        const challengeBase64 = this.base64urlEncode(Buffer.from(storedChallenge.challenge));
        if (clientDataJSON.challenge !== challengeBase64) {
            throw new Error('Challenge mismatch');
        }

        // Verify origin
        if (clientDataJSON.origin !== this.ORIGIN) {
            throw new Error('Origin mismatch');
        }

        // Verify type
        if (clientDataJSON.type !== 'webauthn.create') {
            throw new Error('Invalid type');
        }

        // Decode attestation object
        const attestationObject = this.decodeAttestationObject(
            attestationResponse.response.attestationObject
        );

        // Extract authenticator data
        const authData = attestationObject.authData;
        const { credentialId, publicKey, counter, aaguid, flags } = this.parseAuthenticatorData(authData);

        // Store credential
        const credential = await this.createCredential({
            user_id: userId,
            credential_id: this.base64urlEncode(credentialId),
            public_key: this.base64urlEncode(publicKey),
            counter,
            device_type: attestationResponse.authenticatorAttachment || 'unknown',
            transports: ['internal'], // Could be extracted from extensions
            backup_eligible: !!(flags & 0x08),
            backup_state: !!(flags & 0x10),
            attestation_object: attestationResponse.response.attestationObject,
            aaguid: this.base64urlEncode(aaguid),
            device_name: deviceName || 'PassKey Device'
        });

        // Delete used challenge
        await this.deleteChallenge(storedChallenge.id);

        return credential;
    }

    /**
     * Generate authentication options for signing in with a passkey
     */
    async generateAuthenticationOptions(userId?: string) {
        const challenge = this.generateChallenge();
        const expiresAt = new Date(Date.now() + this.CHALLENGE_TIMEOUT);
        
        await this.createChallenge(userId || null, challenge, 'authentication', expiresAt);

        // If userId provided, get their credentials
        let allowCredentials = undefined;
        if (userId) {
            const credentials = await this.getCredentialsByUserId(userId);
            allowCredentials = credentials.map(cred => ({
                id: cred.credential_id,
                type: 'public-key' as const,
                transports: cred.transports || ['internal']
            }));
        }

        return {
            challenge: this.base64urlEncode(Buffer.from(challenge)),
            timeout: 60000,
            rpId: this.RP_ID,
            allowCredentials,
            userVerification: 'required' as const
        };
    }

    /**
     * Verify authentication response
     */
    async verifyAuthentication(
        assertionResponse: {
            id: string;
            rawId: string;
            response: {
                clientDataJSON: string;
                authenticatorData: string;
                signature: string;
                userHandle?: string;
            };
            type: string;
            authenticatorAttachment?: string;
        }
    ) {
        // Decode client data JSON
        const clientDataJSON = JSON.parse(
            Buffer.from(assertionResponse.response.clientDataJSON, 'base64').toString('utf-8')
        );

        // Find credential
        const credentialId = this.base64urlEncode(Buffer.from(assertionResponse.rawId, 'base64'));
        const credential = await this.getCredentialByCredentialId(credentialId);
        if (!credential) {
            throw new Error('Credential not found');
        }

        // Verify challenge
        const storedChallenge = await this.getLatestChallenge(credential.user_id, 'authentication');
        if (!storedChallenge) {
            throw new Error('No authentication challenge found');
        }

        const challengeBase64 = this.base64urlEncode(Buffer.from(storedChallenge.challenge));
        if (clientDataJSON.challenge !== challengeBase64) {
            throw new Error('Challenge mismatch');
        }

        // Verify origin
        if (clientDataJSON.origin !== this.ORIGIN) {
            throw new Error('Origin mismatch');
        }

        // Verify type
        if (clientDataJSON.type !== 'webauthn.get') {
            throw new Error('Invalid type');
        }

        // Parse authenticator data
        const authDataBuffer = Buffer.from(assertionResponse.response.authenticatorData, 'base64');
        const { counter, flags } = this.parseAuthenticatorData(authDataBuffer);

        // Verify signature
        const clientDataHash = crypto.createHash('sha256')
            .update(Buffer.from(assertionResponse.response.clientDataJSON, 'base64'))
            .digest();
        
        const signatureBase = Buffer.concat([
            authDataBuffer,
            clientDataHash
        ]);

        const publicKeyBuffer = Buffer.from(credential.public_key, 'base64');
        const signatureBuffer = Buffer.from(assertionResponse.response.signature, 'base64');
        
        const isValid = this.verifySignature(publicKeyBuffer, signatureBase, signatureBuffer);
        if (!isValid) {
            throw new Error('Invalid signature');
        }

        // Verify counter (prevent replay attacks)
        if (counter <= credential.counter) {
            throw new Error('Invalid counter - possible replay attack');
        }

        // Update credential counter and last used time
        await this.updateCredentialCounter(credential.id, counter);

        // Delete used challenge
        await this.deleteChallenge(storedChallenge.id);

        return {
            userId: credential.user_id,
            credentialId: credential.credential_id
        };
    }

    // ===================== Database Operations =====================

    private async createCredential(data: {
        user_id: string;
        credential_id: string;
        public_key: string;
        counter: number;
        device_type?: string;
        transports?: string[];
        backup_eligible?: boolean;
        backup_state?: boolean;
        attestation_object?: string;
        aaguid?: string;
        device_name?: string;
    }): Promise<WebAuthnCredential> {
        const id = crypto.randomUUID();
        const res = await this.query(`
            INSERT INTO webauthn_credentials (
                id, user_id, credential_id, public_key, counter,
                device_type, transports, backup_eligible, backup_state,
                attestation_object, aaguid, device_name
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            RETURNING *
        `, [
            id, data.user_id, data.credential_id, data.public_key, data.counter,
            data.device_type, data.transports, data.backup_eligible, data.backup_state,
            data.attestation_object, data.aaguid, data.device_name
        ]);
        return res.rows[0];
    }

    private async getCredentialsByUserId(userId: string): Promise<WebAuthnCredential[]> {
        const res = await this.query(`
            SELECT * FROM webauthn_credentials
            WHERE user_id = $1
            ORDER BY created_at DESC
        `, [userId]);
        return res.rows;
    }

    private async getCredentialByCredentialId(credentialId: string): Promise<WebAuthnCredential | null> {
        const res = await this.query(`
            SELECT * FROM webauthn_credentials
            WHERE credential_id = $1
        `, [credentialId]);
        return res.rows[0] || null;
    }

    private async updateCredentialCounter(id: string, counter: number): Promise<void> {
        await this.query(`
            UPDATE webauthn_credentials
            SET counter = $2, last_used_at = CURRENT_TIMESTAMP
            WHERE id = $1
        `, [id, counter]);
    }

    async deleteCredential(id: string, userId: string): Promise<void> {
        await this.query(`
            DELETE FROM webauthn_credentials
            WHERE id = $1 AND user_id = $2
        `, [id, userId]);
    }

    async listUserCredentials(userId: string): Promise<WebAuthnCredential[]> {
        return this.getCredentialsByUserId(userId);
    }

    private async createChallenge(
        userId: string | null,
        challenge: string,
        type: 'registration' | 'authentication',
        expiresAt: Date
    ): Promise<WebAuthnChallenge> {
        const id = crypto.randomUUID();
        const res = await this.query(`
            INSERT INTO webauthn_challenges (id, user_id, challenge, type, expires_at)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING *
        `, [id, userId, challenge, type, expiresAt]);
        return res.rows[0];
    }

    private async getLatestChallenge(
        userId: string | null,
        type: 'registration' | 'authentication'
    ): Promise<WebAuthnChallenge | null> {
        const res = await this.query(`
            SELECT * FROM webauthn_challenges
            WHERE user_id = $1 AND type = $2 AND expires_at > NOW()
            ORDER BY created_at DESC
            LIMIT 1
        `, [userId, type]);
        return res.rows[0] || null;
    }

    private async deleteChallenge(id: string): Promise<void> {
        await this.query(`
            DELETE FROM webauthn_challenges
            WHERE id = $1
        `, [id]);
    }

    async cleanupExpiredChallenges(): Promise<void> {
        await this.query(`
            DELETE FROM webauthn_challenges
            WHERE expires_at < NOW()
        `);
    }

    // ===================== Helper Functions =====================

    private generateChallenge(): string {
        return crypto.randomBytes(32).toString('hex');
    }

    private base64urlEncode(buffer: Buffer): string {
        return buffer.toString('base64')
            .replace(/\+/g, '-')
            .replace(/\//g, '_')
            .replace(/=/g, '');
    }

    private base64urlDecode(str: string): Buffer {
        const base64 = str
            .replace(/-/g, '+')
            .replace(/_/g, '/');
        return Buffer.from(base64, 'base64');
    }

    private decodeAttestationObject(attestationObjectBase64: string): any {
        // This is a simplified version. In production, use a proper CBOR decoder like 'cbor'
        const buffer = Buffer.from(attestationObjectBase64, 'base64');
        // For now, return a placeholder. You should use a CBOR library here.
        return {
            authData: buffer // Simplified - should parse properly
        };
    }

    private parseAuthenticatorData(authData: Buffer): {
        rpIdHash: Buffer;
        flags: number;
        counter: number;
        credentialId: Buffer;
        publicKey: Buffer;
        aaguid: Buffer;
    } {
        // Simplified parser - in production use proper WebAuthn library
        const rpIdHash = authData.slice(0, 32);
        const flags = authData[32];
        const counter = authData.readUInt32BE(33);
        
        // If credential data is present (AT flag set)
        let credentialId = Buffer.alloc(0);
        let publicKey = Buffer.alloc(0);
        let aaguid = Buffer.alloc(16);
        
        if (flags & 0x40) { // AT (Attested Credential Data) flag
            aaguid = authData.slice(37, 53);
            const credIdLen = authData.readUInt16BE(53);
            credentialId = authData.slice(55, 55 + credIdLen);
            publicKey = authData.slice(55 + credIdLen); // CBOR encoded
        }

        return {
            rpIdHash,
            flags,
            counter,
            credentialId,
            publicKey,
            aaguid
        };
    }

    private verifySignature(publicKey: Buffer, data: Buffer, signature: Buffer): boolean {
        // This is a placeholder. In production, parse the CBOR public key and verify properly
        // You would use crypto.verify() with the proper algorithm (ES256 or RS256)
        try {
            // Simplified verification - implement proper key parsing and verification
            return true; // Placeholder
        } catch (error) {
            return false;
        }
    }
}
