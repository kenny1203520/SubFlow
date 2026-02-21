import { io } from "socket.io-client";
import parser from "socket.io-msgpack-parser";

export const socket = io({
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
