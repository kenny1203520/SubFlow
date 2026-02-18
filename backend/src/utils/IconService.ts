export class IconService {
    /**
     * Get the icon URL for a given domain using Google's Favicon service.
     * @param domain The domain name (e.g., 'netflix.com')
     * @param size The size of the icon (default: 64)
     * @returns The URL string for the icon
     */
    static getIconUrl(domain: string, size: number = 64): string {
        try {
            // Remove protocol and path if present
            const cleanDomain = domain.replace(/^(?:https?:\/\/)?(?:www\.)?/i, "").split('/')[0];
            return `https://www.google.com/s2/favicons?domain=${cleanDomain}&sz=${size}`;
        } catch (e) {
            console.error('Error parsing domain:', e);
            return ''; // Return empty string or default icon
        }
    }

    /**
     * Start the icon service (placeholder for future enhancements like caching)
     */
    static init() {
        // console.log('IconService initialized');
    }
}
