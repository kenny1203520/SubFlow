import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
// import helmet from 'helmet';

import authRoutes from './routes/auth';
import groupRoutes from './routes/groups';
import expenseRoutes from './routes/expenses';
import subscriptionRoutes from './routes/subscriptions';
import { verifySession } from './middleware/auth';

dotenv.config();

const app = express();
const port = process.env.PORT || 3000;

// app.use(helmet());
app.use(cors({
    origin: "http://localhost:5173",
    credentials: true
}));
app.use(express.json());
app.use(verifySession);

app.use("/auth", authRoutes);
app.use("/groups", groupRoutes);
app.use("/expenses", expenseRoutes);
app.use("/subscriptions", subscriptionRoutes);

app.get('/', (req, res) => {
    res.send('SubFlow API is running');
});

app.listen(port, () => {
    console.log(`Server running at http://localhost:${port}`);
});