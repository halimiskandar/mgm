# MyGreenMarket – Eco-Friendly E‑Commerce Backend (Golang)

MGM is an **eco‑friendly e‑commerce backend** that powers a marketplace for sustainable products.  
It is built in **Golang** using **Echo**, **PostgreSQL**, and a clean, modular architecture.

---

## Vision

- Make it **easy to discover eco‑friendly products**
- Provide **transparent information** about materials, packaging, and sustainability
- Enable **fast, safe, and scalable** shopping experiences
- Support **personalised recommendations** for eco‑conscious users

---

## High‑Level Architecture

```text
Frontend (Web / Mobile)
        │  HTTPS / JSON
        ▼
┌─────────────────────────────┐
│      Golang API Server      │
│        (Echo v4)            │
├───────────────┬─────────────┤
│   Middleware   │ Controllers│
├───────────────┼─────────────┤
│       Services / Usecases   │
│  (Users, Products, Orders…) │
├───────────────┼─────────────┤
│      Repositories (Postgres)│
└───────────────▼─────────────┘
              PostgreSQL
```

- **Echo** handles routing & HTTP concerns
- **Middleware** adds auth, logging, CORS, and request tracing
- **Services** (business layer) implement use‑cases
- **Repositories** wrap direct DB access (PostgreSQL)
- **Domain models** live in `domain/` and are shared across layers

---

## Project Structure

```text
app/
  echo-server/
    main.go                
    container
    router/router.go       
    groups & middleware binding
    metrics/                
    endpoint integration

business/
  bandit/                   
  bandit recommender (LinUCB)
  category/                
  service
  mockreco/                 
  fallback recommendations
  orders/                  
  payments/                 
  wallet service
  product/                  
  catalog service
  user/                     
  registration, login, verification

domain/
  user.go                 # User entity
  product.go              # Product entity (eco‑friendly catalog)
  category.go             # Category entity
  orders.go               # Orders entity
  payments.go             # Payments & topup entities
  bandit.go               # bandit events, recommendation model

internal/
  rest/                   # HTTP handlers (Echo)
  middleware/             # auth, admin‑only, self-or-admin, etc.
  repository/             # DB implementations
    postgres/             # PostgreSQL repositories
    redis/                # Redis token repository
    notification/         # Email notification (Mailjet)
    xendit/               # Payment gateway integration

pkg/
  config/                 # config loader (env, struct)
  database/               
    postgres.go           # PostgreSQL connection
    redis/                # Redis client initialization
  logger/                 # structured logger
  metrics/                # bandit & app metrics
  response/               # JSON response helpers
  utils/                  # JWT + password hashing helpers

sql/
  ddl.sql                 # database schema
  dml.sql                 # seed / sample data
```

---

## Data Model (ERD Summary)

**Users** (`users`)
- `id` (PK, uint)
- `full_name`
- `email` (unique)
- `password` (hashed)
- `role` (`customer`, `admin`, …)
- `is_verified` (email verification)
- `wallet` (numeric balance)
- `created_at`, `updated_at`, `deleted_at`

**Categories** (`categories`)
- `category_id` (PK)
- `product_category` (e.g. “Eco Cleaning”, “Organic Food”)
- `created_at`

**Products** (`products`)
- `id` (PK, identity)
- `product_id`, `product_skuid`
- `is_green_tag` (boolean flag)
- `product_name`
- `product_category`
- `unit`
- `normal_price`, `sale_price`, `discount`
- `quantity` (stock)
- `created_at`

**Orders** (`orders`)
- `id` (PK)
- `user_id` (FK → users)
- `product_id` (FK → products)
- `quantity`
- `price_each`, `subtotal`
- `order_status` (e.g. `pending`, `paid`, `cancelled`)
- `payment_method`
- `created_at`, `updated_at`

**Payments** (`payments`)
- `id` (PK)
- `user_id` (FK → users)
- `order_id` (nullable FK → orders)
- `payment_type` (e.g. `ORDER`, `TOPUP`)
- `payment_status` (`PENDING`, `PAID`, …)
- `payment_method`
- `created_at`

**TopUp** (`topups` or virtual entity)
- Logical structure to represent wallet top‑up requests:
  - `id`, `user_id`, `amount`, `top_up_link`

**Bandit / Personalisation**
- `bandit_events`: store feedback events (`view`, `click`, `purchase`) for products & slots
- `user_bandit_segments`: store user → segment mapping for exploration policies

---

##  Core Features

###  Users & Authentication
- Email registration with validation
- Email verification via `/users/email-verification/:code`
- JWT‑based login (`/users/login`)
- **Redis token storage** with session metadata (IP, user agent)
- **Secure logout** with token blacklisting
- **Token refresh** mechanism with old token invalidation
- Role‑based access (customer vs admin)
- **Self-or-Admin** access control - users can only access their own data
- Secure password hashing using **bcrypt**

###  Eco‑Friendly Product Catalog
- Product & category listing
- Filterable catalog 
- Green‑tagged items via `is_green_tag`
- Price fields for normal & sale price

###  Orders
- Create order for a given product and quantity
- Calculate subtotal & store price at time of order
- Track order status & payment method
- Auth‑protected: user can only see / delete their own orders

###  Wallet & Payments
- Wallet balance stored at user level
- Top‑up endpoint generating Xendit payment link
- Payment & top‑up history via `payments` table
- Xendit webhook handler to confirm & apply wallet credit
- Success callback endpoint (`PaidResponse`) for UI

###  Contextual Bandit Recommender
- LinUCB‑based bandit implementation over n‑dimensional feature vectors
- Tracks events (impressions, clicks, conversions) as `BanditEvent`
- `Recommend` endpoint to get product recommendations per slot
- `Feedback` endpoint to send rewards back (clicks / orders)
- Admin routes to configure bandit behaviour & segments
- Offline fallback recommendations module (`mockreco`)

---

##  Tech Stack

| Layer        | Technology                 |
|-------------|---------------------------|
| Language    | Go (Golang)               |
| Framework   | Echo v4                   |
| Database    | PostgreSQL                |
| Cache/Store | Redis (Token Storage)     |
| ORM / DB    | GORM + handcrafted SQL    |
| Auth        | JWT, bcrypt               |
| Metrics     | Prometheus (`/metrics`)   |
| Logging     | Custom structured logger  |
| Payments    | Xendit HTTP integration   |
| Packaging   | Docker (optional)         |

---

##  Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/<your-username>/myGreenMarket.git
cd myGreenMarket/app/echo-server
```

> Adjust the path based on where this backend is placed inside your monorepo.

### 2. Environment Variables

Copy the example config (if provided) or create `.env` with at least:

```env
APP_ENV=local
SERVER_PORT=8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=my_green_market

# Redis Configuration (for token storage)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=supersecretjwt
XENDIT_API_KEY=your_xendit_key_here
```

### 3. Redis Setup

**Option 1: Using Docker (Recommended)**

```bash
# Run Redis container
docker run -d --name redis-dev -p 6379:6379 redis:latest

# Verify Redis is running
docker ps
docker exec -it redis-dev redis-cli ping
# Expected output: PONG
```

### 4. Database Setup

Using `sql/ddl.sql`:

```bash
psql -h localhost -U postgres -d my_green_market -f sql/ddl.sql
```

Optionally seed some data:

```bash
psql -h localhost -U postgres -d my_green_market -f sql/dml.sql
```

### 5. Run the Server

From `app/echo-server`:

```bash
go run main.go
```

The API will be available at:

```text
http://localhost:8080/api/v1
```

Metrics at:

```text
http://localhost:8080/metrics
```

---

## Authentication Flow

1. **Register**
   - `POST /api/v1/users/register`
   - Sends verification code (email implementation handled in user service / notification repository)
2. **Verify Email**
   - `GET /api/v1/users/email-verification/:code`
3. **Login**
   - `POST /api/v1/users/login`
   - Returns JWT access token
   - **Token is stored in Redis** with user metadata (IP address, user agent, expiration)
4. **Use JWT**
   - Pass `Authorization: Bearer <token>` for all protected routes (orders, payments, bandit feedback, etc.)
   - Token is validated against Redis for active sessions
5. **Logout**
   - `POST /api/v1/users/logout`
   - Blacklists token in Redis and removes from active sessions
6. **Refresh Token**
   - `POST /api/v1/users/refresh`
   - Generates new token, blacklists old token, updates Redis storage

---

## Redis Token Storage

### Overview

The application uses **Redis** as a session store for JWT tokens, providing:
- **Real-time token validation** - Tokens are checked against Redis on each request
- **Session management** - Track active sessions per user
- **Token revocation** - Logout immediately invalidates tokens
- **Security audit** - Track IP addresses and user agents per session
- **Token blacklisting** - Prevent reuse of old/refreshed tokens

### Redis Data Structure

**Active Tokens:**
```
Key: token:user:{user_id}
Value: JSON object
{
  "user_id": "123",
  "role": "customer",
  "token": "eyJhbGc...",
  "issued_at": "2025-12-04T10:00:00Z",
  "expires_at": "2025-12-05T10:00:00Z",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0..."
}
TTL: 24 hours
```

**Token Lookup (Reverse Index):**
```
Key: token:lookup:{token_string}
Value: {user_id}
TTL: 24 hours
```

**Blacklisted Tokens:**
```
Key: token:blacklist:{token_string}
Value: "1"
TTL: 24 hours (matches original token expiry)
```

### Token Lifecycle

**Login Flow:**
```
1. User authenticates with email/password
2. JWT token is generated
3. Token + metadata stored in Redis
   - Key: token:user:{user_id}
   - Key: token:lookup:{token}
4. Token returned to client
```

**Protected Request Flow:**
```
1. Client sends: Authorization: Bearer {token}
2. Middleware parses JWT
3. Check Redis: GET token:blacklist:{token}
   - If exists → reject (401)
4. Check Redis: GET token:lookup:{token}
   - If not exists → reject (401)
5. Validate UserID matches
6. Allow request to proceed
```

**Logout Flow:**
```
1. Client sends logout request with token
2. Add to blacklist: SET token:blacklist:{token}
3. Delete from Redis: DEL token:user:{user_id}
4. Delete lookup: DEL token:lookup:{token}
5. Token can no longer be used
```

**Refresh Token Flow:**
```
1. Validate old token from Redis
2. Generate new JWT token
3. Blacklist old token
4. Delete old token from Redis
5. Store new token in Redis
6. Return new token to client
```


## Important Endpoints (API Overview)

Base path: `/api/v1`

### Users

| Method | Path                                  | Description                                | Auth              |
|--------|---------------------------------------|--------------------------------------------|-------------------|
| POST   | `/users/register`                     | Register new user                          | No                |
| GET    | `/users/email-verification/:code`     | Verify email code                          | No                |
| POST   | `/users/login`                        | Login, returns JWT token                   | No                |
| POST   | `/users/logout`                       | Logout account, blacklisting token         | Yes               |
| POST   | `/users/refresh`                      | Refreshing JWT token, return new JWT token | Yes               |
| GET    | `/users/`                             | List All Users                             | Admin only        |
| GET    | `/users/:id`                          | Get user by ID, only user id itself        | Admin/Self-Access |
| PUT    | `/users/:id`                          | Update user                                | Admin/Self-Access |
| DELETE | `/users/:id`                          | Delete user                                | Admin only        |

### Categories

| Method | Path                 | Description               | Auth        |
|--------|----------------------|---------------------------|-------------|
| GET    | `/categories`        | List all categories       | Public      |
| GET    | `/categories/:id`    | Get category by ID        | Public      |
| POST   | `/categories`        | Create category           | Admin only  |
| PUT    | `/categories/:id`    | Update category           | Admin only  |
| DELETE | `/categories/:id`    | Delete category           | Admin only  |

### Products

> Exact paths are defined via `SetupProductRoutes` in `router/router.go` and mirror a standard REST pattern:

| Method | Path              | Description          | Auth        |
|--------|-------------------|----------------------|-------------|
| GET    | `/products`       | List all products    | Public      |
| GET    | `/products/:id`   | Get product details  | Public      |
| POST   | `/products`       | Create product       | Admin only  |
| PUT    | `/products/:id`   | Update product       | Admin only  |
| DELETE | `/products/:id`   | Delete product       | Admin only  |

### Orders

| Method | Path            | Description                      | Auth |
|--------|-----------------|----------------------------------|------|
| POST   | `/orders`       | Create new order                 | Yes  |
| GET    | `/orders`       | Get all orders for current user  | Yes  |
| DELETE | `/orders/:id`   | Delete user’s order              | Yes  |

### Payments & Wallet

| Method | Path                  | Description                              | Auth |
|--------|-----------------------|------------------------------------------|------|
| POST   | `/payments/topup`     | Create top‑up request (Xendit link)      | Yes  |
| GET    | `/payments/success`   | Simple “payment successful” callback     | No   |
| POST   | `/payments/webhook`   | Xendit webhook to confirm payment        | No   |

### Bandit (Recommendations)

Route names follow a structure similar to:

| Method | Path                      | Description                           | Auth |
|--------|---------------------------|---------------------------------------|------|
| GET    | `/bandit/recommend`       | Get recommended products for a slot   | Yes  |
| POST   | `/bandit/feedback`        | Send reward/feedback events           | Yes  |

Admin configuration routes (for configs & segments) are grouped under something like `/bandit/admin/*` and require **admin JWT**.

---

##  Testing the API with Postman

This repo is accompanied by a **Postman collection** (JSON) that you can import directly into Postman:

1. Open **Postman**
2. Click **Import**
3. Select the file: `ecomart_postman_collection.json`
4. Set the `{{base_url}}` variable (e.g. `http://localhost:8080/api/v1`)
5. (Optional) Set `{{auth_token}}` after login to reuse JWT across requests

---

##  Utilities & Cross‑Cutting Concerns

- **JWT utils**: sign & verify tokens, used in `AuthMiddleware`
- **Password hashing**: hash & compare passwords securely
- **Redis token storage**: session management with token validation
- **Logger**: centralised structured logger for requests & errors
- **Metrics**: bandit performance and HTTP metrics for Prometheus
- **Middleware**:
  - `AuthMiddleware()` – basic JWT validation (stateless)
  - `AuthMiddlewareWithRedis()` – JWT validation with Redis session check
  - `AdminOnly()` – restricts access to admin routes
  - `SelfOrAdmin()` – allows users to access their own data, admins can access all

---

## Security Notes

- Never store plain‑text passwords – bcrypt is enforced
- JWT secret must be **strong** and kept private
- **Redis token storage** enables immediate session revocation
- **Token blacklisting** prevents reuse of logged-out or refreshed tokens
- **Session tracking** via IP address and user agent for security auditing
- For production, use **HTTPS** termination in front of this service
- Limit Xendit webhook origin by IP / secret validation
- Consider Redis authentication (REDIS_PASSWORD) in production
- Use Redis Cluster or Sentinel for high availability in production

---

##  Roadmap Ideas

- Product reviews & ratings
- Rich eco‑score calculation system
- Multi‑warehouse inventory & routing
- Full A/B testing dashboard for bandit policies
- Admin UI for categories, products, and configs

---

##  Author & Credits

- **Farhan,Fandi,Halim & Zian** 
- This backend was designed as part of an **eco‑friendly e‑commerce** concept app for hactiv8 project.

