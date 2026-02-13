import axios from 'axios';

// Create a configured axios instance
// In production (BFF mode), baseURL is '/' (relative).
// In development, it points to the backend server.
const http = axios.create({
    baseURL: import.meta.env.PROD ? '/' : 'http://localhost:3000',
    withCredentials: true,
    headers: {
        'Content-Type': 'application/json'
    }
});

export default http;
