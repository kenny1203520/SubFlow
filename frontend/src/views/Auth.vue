<script setup lang="ts">
import { ref } from 'vue';
import axios from 'axios';
import { useRouter } from 'vue-router';

// Configure axios defaults
axios.defaults.baseURL = 'http://localhost:3000';
axios.defaults.withCredentials = true;

const router = useRouter();
const isLogin = ref(true);
const username = ref('');
const email = ref('');
const password = ref('');
const errorMsg = ref('');

const toggleMode = () => {
    isLogin.value = !isLogin.value;
    errorMsg.value = '';
};

const handleSubmit = async () => {
    errorMsg.value = '';
    try {
        if (isLogin.value) {
            await axios.post('/auth/signin', {
                username: username.value,
                password: password.value
            });
            router.push('/dashboard');
        } else {
            await axios.post('/auth/signup', {
                username: username.value,
                email: email.value,
                password: password.value
            });
            // Auto login or ask to login? Let's just switch to login mode or auto-login.
            // For simplicity, let's ask to login
            isLogin.value = true; 
            // OR auto login if the backend set the cookie on signup (which it does in my code)
            router.push('/dashboard');
        }
    } catch (err: any) {
        errorMsg.value = err.response?.data || 'An error occurred';
    }
};
</script>

<template>
<div class="auth-container">
    <div class="auth-box">
        <h2>{{ isLogin ? 'Login to SubFlow' : 'Create Account' }}</h2>
        <form @submit.prevent="handleSubmit">
            <div class="form-group">
                <label>Username</label>
                <input type="text" v-model="username" required minlength="3" />
            </div>
            <div v-if="!isLogin" class="form-group">
                <label>Email</label>
                <input type="email" v-model="email" required />
            </div>
            <div class="form-group">
                <label>Password</label>
                <input type="password" v-model="password" required minlength="6" />
            </div>
            <div v-if="errorMsg" class="error">{{ errorMsg }}</div>
            <button type="submit">{{ isLogin ? 'Login' : 'Sign Up' }}</button>
        </form>
        <p @click="toggleMode" class="toggle-link">
            {{ isLogin ? 'Need an account? Sign up' : 'Already have an account? Login' }}
        </p>
    </div>
</div>
</template>

<style scoped>
.auth-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    background-color: #f0f2f5;
}
.auth-box {
    background: white;
    padding: 2rem;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
    width: 100%;
    max-width: 400px;
}
.form-group {
    margin-bottom: 1rem;
}
.form-group label {
    display: block;
    margin-bottom: 0.5rem;
    color: #333;
}
.form-group input {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
}
button {
    width: 100%;
    padding: 0.75rem;
    background-color: #4CAF50;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    cursor: pointer;
    margin-top: 1rem;
}
button:hover {
    background-color: #45a049;
}
.toggle-link {
    text-align: center;
    margin-top: 1rem;
    color: #666;
    cursor: pointer;
    font-size: 0.9rem;
}
.toggle-link:hover {
    text-decoration: underline;
}
.error {
    color: red;
    font-size: 0.9rem;
    margin-top: 0.5rem;
}
</style>
