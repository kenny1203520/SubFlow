import crypto from 'crypto';
import { Request } from 'express';

/**
 * Device fingerprinting utilities
 * 
 * These functions collect various device attributes to create a unique fingerprint
 * that can help identify devices across sessions, even without cookies.
 */

/**
 * Generate a device fingerprint from request headers and other attributes
 * This is a server-side fingerprint based on information available in the HTTP request
 * 
 * For better accuracy, combine this with client-side fingerprinting using libraries like:
 * - FingerprintJS (https://github.com/fingerprintjs/fingerprintjs)
 * - ClientJS
 */
export function generateDeviceFingerprint(req: Request, clientFingerprint?: string): string {
    if (clientFingerprint) {
        // Use client-provided fingerprint if available (more accurate)
        return clientFingerprint;
    }

    // Fallback to server-side fingerprinting
    const components = [
        req.headers['user-agent'] || '',
        req.headers['accept-language'] || '',
        req.headers['accept-encoding'] || '',
        req.headers['accept'] || '',
        // Note: Don't use IP as primary component as it can change (mobile, VPN, etc.)
    ];

    const fingerprintString = components.join('|');
    return crypto.createHash('sha256').update(fingerprintString).digest('hex');
}

/**
 * Get device fingerprint from request
 * Checks for client-provided fingerprint first, then generates server-side fingerprint
 */
export function getDeviceFingerprint(req: Request): string {
    // Check if client sent a fingerprint (e.g., from FingerprintJS)
    const clientFingerprint = req.headers['x-device-fingerprint'] as string || req.body?.deviceFingerprint;
    return generateDeviceFingerprint(req, clientFingerprint);
}

/**
 * Parse user agent to extract device information
 */
export function parseUserAgent(userAgent: string): {
    browser: string;
    os: string;
    device: string;
    isMobile: boolean;
} {
    const ua = userAgent.toLowerCase();

    // Detect browser
    let browser = 'Unknown Browser';
    if (ua.indexOf('firefox') > -1) browser = 'Firefox';
    else if (ua.indexOf('samsungbrowser') > -1) browser = 'Samsung Internet';
    else if (ua.indexOf('opera') > -1 || ua.indexOf('opr') > -1) browser = 'Opera';
    else if (ua.indexOf('trident') > -1) browser = 'Internet Explorer';
    else if (ua.indexOf('edg') > -1) browser = 'Edge';
    else if (ua.indexOf('chrome') > -1) browser = 'Chrome';
    else if (ua.indexOf('safari') > -1) browser = 'Safari';

    // Detect OS
    let os = 'Unknown OS';
    if (ua.indexOf('win') > -1) os = 'Windows';
    else if (ua.indexOf('mac') > -1) os = 'MacOS';
    else if (ua.indexOf('linux') > -1) os = 'Linux';
    else if (ua.indexOf('android') > -1) os = 'Android';
    else if (ua.indexOf('like mac') > -1) os = 'iOS';

    // Detect device type
    let device = 'Desktop';
    let isMobile = false;
    if (ua.indexOf('mobile') > -1 || ua.indexOf('android') > -1) {
        device = 'Mobile';
        isMobile = true;
    } else if (ua.indexOf('tablet') > -1 || ua.indexOf('ipad') > -1) {
        device = 'Tablet';
        isMobile = true;
    }

    return { browser, os, device, isMobile };
}

/**
 * Generate a friendly device name from user agent
 */
export function generateDeviceName(userAgent: string): string {
    const { browser, os, device } = parseUserAgent(userAgent);
    return `${browser} on ${os} (${device})`;
}

/**
 * Generate a secure device token
 * This token is stored in a cookie and used to identify returning devices
 */
export function generateDeviceToken(): string {
    return crypto.randomBytes(32).toString('hex');
}

/**
 * Compare two device fingerprints with fuzzy matching
 * Returns a similarity score (0-1)
 * Useful for detecting if a device has changed slightly (e.g., browser update)
 */
export function compareFingerprints(fp1: string, fp2: string): number {
    if (fp1 === fp2) return 1.0;
    
    // Simple Hamming distance for hex strings
    if (fp1.length !== fp2.length) return 0;
    
    let matches = 0;
    for (let i = 0; i < fp1.length; i++) {
        if (fp1[i] === fp2[i]) matches++;
    }
    
    return matches / fp1.length;
}

/**
 * Extract IP address from request, considering proxies
 */
export function getClientIpAddress(req: Request): string {
    return (
        req.headers['x-forwarded-for'] as string ||
        req.headers['x-real-ip'] as string ||
        req.socket.remoteAddress ||
        ''
    ).split(',')[0].trim();
}

/**
 * Check if IP address has changed significantly (different subnet/country)
 * This is a simplified check - in production, use a GeoIP service
 */
export function hasIpAddressChanged(oldIp: string, newIp: string): boolean {
    // Simple check: compare first 3 octets (same /24 subnet)
    const oldParts = oldIp.split('.');
    const newParts = newIp.split('.');
    
    if (oldParts.length !== 4 || newParts.length !== 4) {
        return true; // If not IPv4, consider as changed
    }
    
    // Check if first 3 octets match (same /24 subnet)
    return !(
        oldParts[0] === newParts[0] &&
        oldParts[1] === newParts[1] &&
        oldParts[2] === newParts[2]
    );
}

/**
 * Get device token from request cookies or headers
 */
export function getDeviceTokenFromRequest(req: Request): string | null {
    // Check cookie first
    if (req.cookies?.device_token) {
        return req.cookies.device_token;
    }
    
    // Check header
    if (req.headers['x-device-token']) {
        return req.headers['x-device-token'] as string;
    }
    
    return null;
}

/**
 * Risk scoring for login attempts
 * Returns a risk score (0-100) based on various factors
 */
export function calculateLoginRiskScore(factors: {
    isNewDevice: boolean;
    isNewLocation: boolean;
    fingerprintSimilarity: number;
    failedAttemptsRecently: number;
    accountAge: number; // in days
    timeSinceLastLogin: number; // in hours
}): number {
    let score = 0;

    // New device adds risk
    if (factors.isNewDevice) score += 30;
    
    // New location adds risk
    if (factors.isNewLocation) score += 25;
    
    // Low fingerprint similarity adds risk
    if (factors.fingerprintSimilarity < 0.8) score += 20;
    
    // Recent failed attempts add risk
    score += Math.min(factors.failedAttemptsRecently * 5, 25);
    
    // Very new accounts are riskier
    if (factors.accountAge < 1) score += 15;
    
    // Very long time since last login adds risk
    if (factors.timeSinceLastLogin > 720) score += 10; // 30 days
    
    return Math.min(score, 100);
}

/**
 * Determine if 2FA should be required based on risk score
 */
export function shouldRequire2FA(riskScore: number, has2FAEnabled: boolean): boolean {
    if (!has2FAEnabled) return false;
    
    // Require 2FA for medium to high risk logins
    return riskScore >= 40;
}
