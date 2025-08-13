package main

import (
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

// jwtSecret is the secret key used to sign JWTs. It is initialized from the JWT_SECRET
// environment variable or defaults to "secret" when unset.
var jwtSecret = func() string {
    if v := os.Getenv("JWT_SECRET"); v != "" {
        return v
    }
    return "secret"
}()

// hashPassword returns a bcrypt hashed representation of the plain password.
func hashPassword(password string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

// checkPassword compares a bcrypt hashed password with its possible plaintext equivalent.
// Returns nil if they match or an error otherwise.
func checkPassword(hash string, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// generateJWT creates a signed JSON Web Token for a given user ID with a standard expiry of 24 hours.
func generateJWT(userID int64) (string, error) {
    claims := jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(jwtSecret))
}
