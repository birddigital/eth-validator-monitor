# Testing Authentication Endpoints

This document provides comprehensive test cases for the `/api/auth/register` and `/api/auth/login` endpoints.

## Endpoints

### POST /api/auth/register
Creates a new user account with validation.

### POST /api/auth/login
Authenticates an existing user.

## Test Strategy

Test with curl or Postman/Insomnia to send valid and invalid payloads. Verify correct status codes and structured error responses.

## 1. Registration Endpoint Tests

### Test 1.1: Successful Registration

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Password123!",
    "confirmPassword": "Password123!"
  }'
```

**Expected Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "testuser",
  "email": "test@example.com",
  "roles": ["user"]
}
```

### Test 1.2: Invalid Email Format

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "invalid-email",
    "password": "Password123!",
    "confirmPassword": "Password123!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "email": "invalid email format"
  }
}
```

### Test 1.3: Weak Password (No Uppercase)

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123!",
    "confirmPassword": "password123!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "password": "password must contain uppercase, lowercase, number, and special character"
  }
}
```

### Test 1.4: Weak Password (Too Short)

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Pass1!",
    "confirmPassword": "Pass1!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "password": "password must be at least 8 characters long"
  }
}
```

### Test 1.5: Weak Password (No Special Character)

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Password123",
    "confirmPassword": "Password123"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "password": "password must contain uppercase, lowercase, number, and special character"
  }
}
```

### Test 1.6: Password Mismatch

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "Password123!",
    "confirmPassword": "DifferentPassword123!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "confirmPassword": "passwords do not match"
  }
}
```

### Test 1.7: Username Too Short

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "ab",
    "email": "test@example.com",
    "password": "Password123!",
    "confirmPassword": "Password123!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "username": "username must be at least 3 characters long"
  }
}
```

### Test 1.8: Email Already Exists

**Request (after successful registration with same email):**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "differentuser",
    "email": "test@example.com",
    "password": "Password123!",
    "confirmPassword": "Password123!"
  }'
```

**Expected Response (409 Conflict):**
```json
{
  "error": "Conflict",
  "message": "User already exists",
  "fields": {
    "username": "username or email already exists"
  }
}
```

### Test 1.9: Username Already Exists

**Request (after successful registration with same username):**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "different@example.com",
    "password": "Password123!",
    "confirmPassword": "Password123!"
  }'
```

**Expected Response (409 Conflict):**
```json
{
  "error": "Conflict",
  "message": "User already exists",
  "fields": {
    "username": "username or email already exists"
  }
}
```

### Test 1.10: Multiple Validation Errors

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "ab",
    "email": "invalid-email",
    "password": "weak",
    "confirmPassword": "different"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "username": "username must be at least 3 characters long",
    "email": "invalid email format",
    "password": "password must be at least 8 characters long",
    "confirmPassword": "passwords do not match"
  }
}
```

### Test 1.11: Missing Fields

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "",
    "email": "",
    "password": "",
    "confirmPassword": ""
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "username": "username is required",
    "email": "email is required",
    "password": "password must be at least 8 characters long"
  }
}
```

### Test 1.12: Invalid JSON

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Invalid request body",
  "fields": {
    "body": "invalid JSON"
  }
}
```

## 2. Login Endpoint Tests

### Test 2.1: Successful Login

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "Password123!"
  }'
```

**Expected Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "testuser",
  "email": "test@example.com",
  "roles": ["user"]
}
```

**Note:** A session cookie should be set in the response headers.

### Test 2.2: Invalid Username

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nonexistent",
    "password": "Password123!"
  }'
```

**Expected Response (401 Unauthorized):**
```json
{
  "error": "Unauthorized",
  "message": "Invalid credentials",
  "fields": {
    "credentials": "invalid username or password"
  }
}
```

### Test 2.3: Invalid Password

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "WrongPassword123!"
  }'
```

**Expected Response (401 Unauthorized):**
```json
{
  "error": "Unauthorized",
  "message": "Invalid credentials",
  "fields": {
    "credentials": "invalid username or password"
  }
}
```

### Test 2.4: Missing Username

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "",
    "password": "Password123!"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "username": "username is required"
  }
}
```

### Test 2.5: Missing Password

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": ""
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "password": "password is required"
  }
}
```

### Test 2.6: Missing Both Fields

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "",
    "password": ""
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Validation failed",
  "fields": {
    "username": "username is required",
    "password": "password is required"
  }
}
```

### Test 2.7: Invalid JSON

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

**Expected Response (400 Bad Request):**
```json
{
  "error": "Bad Request",
  "message": "Invalid request body",
  "fields": {
    "body": "invalid JSON"
  }
}
```

## 3. Session Management Tests

### Test 3.1: Get Current User (Authenticated)

**Request:**
```bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Cookie: session=<session_cookie_from_login>"
```

**Expected Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "testuser",
  "email": "test@example.com",
  "roles": ["user"]
}
```

### Test 3.2: Get Current User (Unauthenticated)

**Request:**
```bash
curl -X GET http://localhost:8080/api/auth/me
```

**Expected Response (401 Unauthorized):**
```json
{
  "error": "Not authenticated"
}
```

### Test 3.3: Logout

**Request:**
```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Cookie: session=<session_cookie_from_login>"
```

**Expected Response (200 OK):**
```json
{
  "message": "Logged out successfully"
}
```

## Summary of Validation Rules

### Email Validation
- **Required**: Must not be empty
- **Format**: Must match email regex pattern `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
- **Uniqueness**: Must not already exist in database

### Username Validation
- **Required**: Must not be empty
- **Min Length**: 3 characters
- **Max Length**: 255 characters
- **Uniqueness**: Must not already exist in database

### Password Validation
- **Required**: Must not be empty
- **Min Length**: 8 characters
- **Complexity**: Must contain:
  - At least 1 uppercase letter
  - At least 1 lowercase letter
  - At least 1 number
  - At least 1 special character (punctuation or symbol)
- **Confirmation**: Must match `confirmPassword` field

## Status Codes

| Code | Status | Description |
|------|--------|-------------|
| 200 | OK | Successful login or logout |
| 201 | Created | Successful registration |
| 400 | Bad Request | Validation errors or invalid JSON |
| 401 | Unauthorized | Invalid credentials or not authenticated |
| 409 | Conflict | Username or email already exists |
| 500 | Internal Server Error | Server error (database, session, etc.) |

## Error Response Format

All error responses follow this structured format:

```json
{
  "error": "HTTP Status Text",
  "message": "Human-readable message",
  "fields": {
    "fieldName": "Specific error for this field"
  }
}
```

- `error`: HTTP status text (e.g., "Bad Request", "Unauthorized")
- `message`: Overall error description
- `fields`: Object with field-specific errors (optional, present for validation errors)

## Security Considerations

1. **Password Hashing**: Passwords are hashed using bcrypt with cost factor 12
2. **Session Security**: Sessions are stored in Redis with secure cookies
3. **Generic Error Messages**: Login errors don't reveal whether username or password is incorrect
4. **Email Uniqueness**: Checked before password hashing to prevent timing attacks
5. **No User Enumeration**: Error messages don't reveal if a user exists during login

## Testing Workflow

1. Start the server: `go run cmd/server/main.go`
2. Ensure PostgreSQL and Redis are running
3. Run database migrations
4. Execute test cases in order
5. Verify status codes and response structure
6. Check that sessions are created on successful login/register
7. Verify validation errors show field-level details

## Automated Testing

For automated testing, consider writing integration tests using:

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestRegisterEndpoint(t *testing.T) {
    // Create test server
    // Send requests
    // Assert responses
}
```

See `/go-crypto` consultation in CLAUDE.md for example test code structure.
