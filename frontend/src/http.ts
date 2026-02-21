import axios from 'axios';

// Create a configured axios instance
// In production (BFF mode), baseURL is '/' (relative).
// In development, it points to the backend server.
const http = axios.create({
    baseURL: '/',
    withCredentials: true,
    headers: {
        'Content-Type': 'application/json'
    }
});

export default http;
