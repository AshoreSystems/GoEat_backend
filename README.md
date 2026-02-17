# GoEat Backend API

A food delivery platform backend built with Go, featuring restaurant management, order processing, and payment integration.

## Features

- **User Authentication**: Customer and restaurant login with JWT tokens
- **Restaurant Management**: Menu items, business hours, order management
- **Order Processing**: Real-time order tracking and status updates
- **Payment Integration**: Stripe payment processing with refund support
- **Email Notifications**: Order confirmations and updates
- **Address Management**: Customer delivery addresses with default selection

## API Endpoints

**Base URL:** `https://your-server.com` or `http://your-server-ip:8080`

### Payment
- `POST /create-payment-intent` - Create Stripe payment intent with user validation

### Orders
- `POST /place-order` - Place new order
- `GET /customer-orders` - Get customer order history
- `POST /cancel-order` - Cancel order with refund
- `GET /restaurant-orders` - Get restaurant orders by status

### Authentication
- User status validation (active/inactive/blocked/pending)
- JWT token-based authentication

## Recent Updates

### Create Payment Intent Enhancement
- Added `user_id` parameter validation
- Customer table lookup with `login_id` verification
- Account status check (only active users can create payments)
- Proper JSON error responses
- Enhanced security with multi-table validation

## Database Schema

### Key Tables
- `customer` - Customer information with `login_id` reference
- `login` - Authentication with status enum('active','inactive','blocked','pending')
- `restaurants` - Restaurant details and business hours
- `tbl_orders` - Order management
- `tbl_payment_transactions` - Payment tracking

## Prerequisites

- Go 1.19 or higher
- MySQL 8.0 or higher
- Git
- Stripe account for payment processing
- SMTP server for email notifications

## Installation

### 1. Clone Repository
```bash
git clone https://github.com/your-username/GoEat_backend.git
cd GoEat_backend
```

### 2. Install Dependencies
```bash
go mod download
go mod tidy
```

### 3. Database Setup
```sql
-- Create database
CREATE DATABASE goeat_db;
USE goeat_db;

-- Import your SQL schema file
source schema.sql;
```

### 4. Environment Configuration
Create `.env` file in root directory:
```env
# Database Configuration
DB_HOST=your-server-ip
DB_PORT=3306
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=goeat_db

# Stripe Configuration
STRIPE_PK=pk_test_your_stripe_publishable_key
STRIPE_SK=sk_test_your_stripe_secret_key

# JWT Configuration
JWT_SECRET=your_jwt_secret_key

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASSWORD=your_app_password

# Server Configuration
PORT=8080
SERVER_HOST=0.0.0.0
```

### 5. Run Application
```bash
# Development mode
go run main.go

# Build and run
go build -o goeat-api
./goeat-api
```

### 6. Verify Installation
Test the API:
```bash
curl http://your-server-ip:8080/health
```

## Setup

### Database Tables
Ensure these tables exist in your MySQL database:
- `customer` - Customer profiles
- `login` - Authentication records
- `restaurants` - Restaurant information
- `menu_items` - Restaurant menu items
- `tbl_orders` - Order records
- `tbl_order_items` - Order item details
- `tbl_payment_transactions` - Payment records
- `customer_delivery_addresses` - Customer addresses
- `tbl_ratings_reviews` - Customer reviews

### API Testing
Use tools like Postman or curl to test endpoints:
```bash
# Test payment intent creation
curl -X POST http://your-server-ip:8080/create-payment-intent \
  -F "user_id=1" \
  -F "amount=25.50" \
  -F "restaurant_id=1"
```

## Deployment

### Production Build
```bash
# Build for production
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goeat-api .

# Run with production settings
GIN_MODE=release ./goeat-api
```

### Docker Deployment
```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goeat-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/goeat-api .
CMD ["./goeat-api"]
```

## Security Features

- User account status validation
- SQL injection prevention with parameterized queries
- JWT token authentication
- Payment intent validation before processing