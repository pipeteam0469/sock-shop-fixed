const express = require('express');
const axios = require('axios');
const helmet = require('helmet');
const rateLimit = require('express-rate-limit');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 8079;

// Security middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
    },
  },
}));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100, // limit each IP to 100 requests per windowMs
  message: { error: 'Too many requests, please try again later.' }
});
app.use(limiter);

// CORS configuration
app.use(cors({
  origin: '*',
  methods: ['GET', 'POST'],
  allowedHeaders: ['Content-Type', 'Authorization']
}));

// View engine
app.set('view engine', 'ejs');
app.set('views', path.join(__dirname, '..', 'views'));
app.use(express.static(path.join(__dirname, '..', 'public')));
app.use(express.json());

// Service hosts (from environment variables)
const CATALOGUE_HOST = process.env.CATALOGUE_HOST || 'catalogue';
const USER_HOST = process.env.USER_HOST || 'user';
const PAYMENT_HOST = process.env.PAYMENT_HOST || 'payment';

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'front-end' });
});

// Home page
app.get('/', async (req, res) => {
  try {
    const catalogueUrl = `http://${CATALOGUE_HOST}:8080/catalogue`;
    const response = await axios.get(catalogueUrl, { timeout: 5000 });
    const products = response.data || [];
    res.render('index', { products: products.slice(0, 12) });
  } catch (error) {
    console.error('Error fetching catalogue:', error.message);
    res.render('index', { products: [] });
  }
});

// Product detail page
app.get('/detail/:id', async (req, res) => {
  try {
    const catalogueUrl = `http://${CATALOGUE_HOST}:8080/catalogue/${req.params.id}`;
    const response = await axios.get(catalogueUrl, { timeout: 5000 });
    res.render('detail', { product: response.data });
  } catch (error) {
    console.error('Error fetching product:', error.message);
    res.status(500).render('error', { message: 'Product not found' });
  }
});

// User login page
app.get('/login', (req, res) => {
  res.render('login');
});

// User login endpoint
app.post('/login', async (req, res) => {
  try {
    const { username, password } = req.body;
    const userUrl = `http://${USER_HOST}:8080/login`;
    const response = await axios.post(userUrl, { username, password }, { timeout: 5000 });
    res.json(response.data);
  } catch (error) {
    console.error('Login error:', error.message);
    res.status(500).json({ error: 'Login failed' });
  }
});

// User register page
app.get('/register', (req, res) => {
  res.render('register');
});

// User register endpoint
app.post('/register', async (req, res) => {
  try {
    const userUrl = `http://${USER_HOST}:8080/register`;
    const response = await axios.post(userUrl, req.body, { timeout: 5000 });
    res.json(response.data);
  } catch (error) {
    console.error('Registration error:', error.message);
    res.status(500).json({ error: 'Registration failed' });
  }
});

// Cart page
app.get('/cart', (req, res) => {
  res.render('cart');
});

// Payment page
app.get('/payment', (req, res) => {
  res.render('payment');
});

// Process payment
app.post('/payment', async (req, res) => {
  try {
    const paymentUrl = `http://${PAYMENT_HOST}:8080/payment`;
    const response = await axios.post(paymentUrl, req.body, { timeout: 5000 });
    res.json(response.data);
  } catch (error) {
    console.error('Payment error:', error.message);
    res.status(500).json({ error: 'Payment failed' });
  }
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Front-end service running on port ${PORT}`);
  console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
  console.log(`Catalogue host: ${CATALOGUE_HOST}`);
  console.log(`User host: ${USER_HOST}`);
  console.log(`Payment host: ${PAYMENT_HOST}`);
});
