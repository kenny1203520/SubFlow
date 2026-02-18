import { io } from "socket.io-client";
import parser from "socket.io-msgpack-parser";

const URL = import.meta.env.PROD ? undefined : "http://localhost:3000";

export const socket = io(URL as string, {
    parser,
    withCredentials: true,
    autoConnect: false // Connect only after login
});

socket.on("connect", () => {
    // console.log("Connected to Socket.IO server");
});

socket.on("connect_error", (err) => {
    console.error("Connection error:", err.message);
});
