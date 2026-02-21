import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import { createProxyMiddleware } from 'http-proxy-middleware';
import history from 'connect-history-api-fallback';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const port = process.env.PORT || 4173;
const backendUrl = process.env.BACKEND_URL || 'http://localhost:3000';

// 1. Proxy middleware configuration
const proxyOptions = {
    target: backendUrl,
    changeOrigin: true,
    xfwd: true,
};

// API, Auth, and Uploads proxying
// We mount the proxy on the root and use a pathFilter to catch multiple prefixes
// This prevents Express from stripping the prefix (e.g. /auth) from the path sent to the backend
app.use(createProxyMiddleware({
    ...proxyOptions,
    pathFilter: ['/api', '/auth', '/uploads', '/socket.io'],
    ws: true, // Handle WebSockets within the same middleware
}));

// 2. SPA Fallback
// Redirects all non-file requests to index.html for Vue Router
app.use(history({
    index: '/index.html',
    verbose: false,
    rewrites: [
        { from: /\/auth/, to: '/auth' }, // Allow auth proxy to bypass history
        { from: /\/api/, to: '/api' },   // Allow api proxy to bypass history
        { from: /\/uploads/, to: '/uploads' }
    ]
}));

// 3. Static files
// Serve the built frontend files from the dist directory
app.use(express.static(path.join(__dirname, 'dist')));

app.listen(port, () => {
    console.log(`BFF Server active at http://localhost:${port}`);
    console.log(`Forwarding API requests to: ${backendUrl}`);
});
